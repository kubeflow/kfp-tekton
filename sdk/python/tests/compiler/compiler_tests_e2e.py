# Copyright 2020-2021 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import logging
import os
import re
import textwrap
import unittest
import yaml

from glob import glob
from kfp_tekton import TektonClient
from packaging import version
from os import environ as env
from subprocess import run, SubprocessError
from time import sleep


# =============================================================================
#  load test settings from environment variables (passed through make)
# =============================================================================

# get the Kubernetes context from the KUBECONFIG env var, override KUBECONFIG
# to target a different Kubernetes cluster
#    KUBECONFIG=/path/to/kube/config sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test KUBECONFIG=/path/to/kube/config
KUBECONFIG = env.get("KUBECONFIG")

# warn the user that the KUBECONFIG variable was not set so the target cluster
# might not be the expected one
if not KUBECONFIG:
    logging.warning("The environment variable 'KUBECONFIG' was not set.")
else:
    logging.warning("KUBECONFIG={}".format(KUBECONFIG))

# set or override the minimum required Tekton Pipeline version, default "v0.20.1":
#    TKN_PIPELINE_MIN_VERSION=v0.20 sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test TKN_PIPELINE_MIN_VERSION=v0.20
TKN_PIPELINE_MIN_VERSION = env.get("TKN_PIPELINE_MIN_VERSION", "v0.20.1")

# let the user know the expected Tekton Pipeline version
if env.get("TKN_PIPELINE_MIN_VERSION"):
    logging.warning("The environment variable 'TKN_PIPELINE_MIN_VERSION' was set to '{}'"
                    .format(TKN_PIPELINE_MIN_VERSION))

# set or override the minimum required Tekton CLI version, default "0.15.0":
#    TKN_CLIENT_MIN_VERSION=0.15 sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test TKN_CLIENT_MIN_VERSION=0.15
TKN_CLIENT_MIN_VERSION = env.get("TKN_CLIENT_MIN_VERSION", "0.15.0")

# let the user know the expected Tekton CLI version
if env.get("TKN_CLIENT_MIN_VERSION"):
    logging.warning("The environment variable 'TKN_CLIENT_MIN_VERSION' was set to '{}'"
                    .format(TKN_CLIENT_MIN_VERSION))

# Temporarily set GENERATE_GOLDEN_E2E_LOGS=True to (re)generate new "golden" log
# files after making code modifications that change the expected log output.
# To (re)generate all "golden" log output files from the command line run:
#    GENERATE_GOLDEN_E2E_LOGS=True sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test GENERATE_GOLDEN_E2E_LOGS=True
GENERATE_GOLDEN_E2E_LOGS = env.get("GENERATE_GOLDEN_E2E_LOGS", "False") == "True"

# let the user know this test run is not performing any verification
if GENERATE_GOLDEN_E2E_LOGS:
    logging.warning(
        "The environment variable 'GENERATE_GOLDEN_E2E_LOGS' was set to 'True'. "
        "Test cases will (re)generate the 'golden' log files instead of verifying "
        "the logs produced by running the Tekton YAML on a Kubernetes cluster.")

# When USE_LOGS_FROM_PREVIOUS_RUN=True, the logs from the previous pipeline run
# will be used for log verification or for regenerating "golden" log files:
#    USE_LOGS_FROM_PREVIOUS_RUN=True sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test USE_LOGS_FROM_PREVIOUS_RUN=True
#
# NOTE: this is problematic since multiple test cases (YAML files) use the same
#   pipelinerun name, so `tkn pipelinerun logs <pipelinerun-name>` will always
#   return the logs of the last test case running a pipeline with that name
#
# TODO: make sure each YAML file has a unique pipelinerun name, i.e.
#   cd sdk/python/tests/compiler/testdata; \
#       (echo "PIPELINE_NAME FILE"; grep -E "^  name: "  *.yaml | \
#           sed -e 's/ name: //g' | sed -e 's/\(.*\): \(.*\)/\2 \1/g' | sort ) | \
#       column -t
#
# TODO: TestCompilerE2E.pr_name_map needs to be built from `tkn pipelinerun list`
#   tkn pipelinerun list -n kubeflow -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | head
#
# ALSO: once we delete pipelineruns after success, logs won't be available after
#   that test execution
#
USE_LOGS_FROM_PREVIOUS_RUN = env.get("USE_LOGS_FROM_PREVIOUS_RUN", "False") == "True"

# let the user know we are using the logs produced during a previous test run
if USE_LOGS_FROM_PREVIOUS_RUN:
    logging.warning(
        "The environment variable 'USE_LOGS_FROM_PREVIOUS_RUN' was set to 'True'. "
        "Test cases will use the logs produced by a prior pipeline execution. "
        "The Tekton YAML will not be verified on a Kubernetes cluster.")

# set INCLUDE_TESTS environment variable to only run the specified E2E tests:
#    INCLUDE_TESTS=test_name1,test_name2 sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test INCLUDE_TESTS=test_name1,test_name2
INCLUDE_TESTS = env.get("INCLUDE_TESTS", "")

# let the user know we are only running specified test
if INCLUDE_TESTS:
    logging.warning("INCLUDE_TESTS={} ".format(INCLUDE_TESTS))

# set EXCLUDE_TESTS environment variable to exclude the specified E2E tests:
#    EXCLUDE_TESTS=test_name1,test_name2 sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test EXCLUDE_TESTS=test_name1,test_name2
EXCLUDE_TESTS = env.get("EXCLUDE_TESTS", "")

# let the user know which tests are excluded
if EXCLUDE_TESTS:
    logging.warning("EXCLUDE_TESTS={} ".format(EXCLUDE_TESTS))


# TODO: delete pipelineruns (and logs) after test run, keep failed runs (and logs)
#   if KEEP_FAILED_PIPELINERUNS=True
# KEEP_FAILED_PIPELINERUNS = env.get("KEEP_FAILED_PIPELINERUNS", "False") == "True"


# Set SLEEP_BETWEEN_TEST_PHASES=<seconds> (default: 5) to increase or decrease
# the sleep time between the test stages of starting a pipelinerun, then first
# attempting to get the pipelinerun status, and lastly to get the pipelinerun
# logs. Increase the sleep for under-powered Kubernetes clusters. The minimal
# recommended configuration for K8s clusters is 4 cores, 2 nodes, 16 GB RAM:
#    SLEEP_BETWEEN_TEST_PHASES=10 sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test SLEEP_BETWEEN_TEST_PHASES=10
SLEEP_BETWEEN_TEST_PHASES = int(env.get("SLEEP_BETWEEN_TEST_PHASES", "5"))

# let the user know this test run is not performing any verification
if env.get("SLEEP_BETWEEN_TEST_PHASES"):
    logging.warning(
        "The environment variable 'SLEEP_BETWEEN_TEST_PHASES' was set to '{}'. "
        "Default is '5' seconds. Increasing this value should improve the test "
        "success rate on a slow Kubernetes cluster.".format(SLEEP_BETWEEN_TEST_PHASES))

# set RERUN_FAILED_TESTS_ONLY=True, to only re-run those E2E tests that failed in
# the previous test run:
#    RERUN_FAILED_TESTS_ONLY=True sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test RERUN_FAILED_TESTS_ONLY=True
RERUN_FAILED_TESTS_ONLY = env.get("RERUN_FAILED_TESTS_ONLY", "False") == "True"

# the file used to keep a record of failed tests
failed_tests_file = os.path.join(os.path.dirname(__file__), ".failed_tests")

# the list of test_names that failed in the previous test run
previously_failed_tests = []

# let the user know we are running previously failed tests only
if RERUN_FAILED_TESTS_ONLY:
    logging.warning("The environment variable 'RERUN_FAILED_TESTS_ONLY' was set to 'True'.")
    if os.path.exists(failed_tests_file):
        with open(failed_tests_file, 'r') as f:
            previously_failed_tests = f.read().splitlines()
        logging.warning(
            "Running previously failed tests only: {}".format(previously_failed_tests))
    else:
        logging.warning("Could not find file {}".format(os.path.abspath(failed_tests_file)))


# =============================================================================
#  non-configurable test settings
# =============================================================================

# ignore pipelines with unpredictable log output or complex prerequisites
# TODO: revisit this list, try to rewrite those Python DSLs in a way that they
#  will produce logs which can be verified. One option would be to keep more
#  than one "golden" log file to match either of the desired possible outputs
ignored_yaml_files = [
    "big_data_passing.yaml",    # does not complete in a reasonable time frame
    "create_component_from_func_component.yaml",  # not a Tekton PipelineRun
    "create_component_from_func.yaml",  # need to investigate, keeps Running
    "katib.yaml",               # service account needs Katib permission, takes too long doing 9 trail runs
    "parallel_join_with_logging.yaml",  # experimental feature, requires S3 (Minio)
    "retry.yaml",               # designed to occasionally fail (randomly) if number of retries exceeded
    "timeout.yaml",             # random failure (by design) ... would need multiple golden log files to compare to
    "tolerations.yaml",         # designed to fail, test show only how to add the toleration to the pod
    "volume.yaml",              # need to rework the credentials part
    "volume_op.yaml",           # need to delete PVC before/after test run
    "volume_snapshot_op.yaml",  # only works on Minikube, K8s alpha feature, requires a feature gate from K8s master

    # the following tests require tekton-pipelines feature-flag data.enable-custom-tasks=true
    #   kubectl patch cm feature-flags -n tekton-pipelines \
    #     -p '{"data":{"disable-home-env-overwrite":"true","disable-working-directory-overwrite":"true", "enable-custom-tasks": "true"}}'
    # TODO: apply the _cr*.yaml files "apiVersion: custom.tekton.dev/v1alpha1, kind: PipelineLoop"
    #   for f in sdk/python/tests/compiler/testdata/*_cr*.yaml; do \
    #     echo "=== ${f} ==="; \
    #     kubectl apply -f "${f}" -n kubeflow && \
    #     echo OK || echo FAILED; \
    #   done
    "conditions_and_loops.yaml",
    "loop_over_lightweight_output.yaml",
    "loop_static.yaml",
    "parallelfor_item_argument_resolving.yaml",
    "withitem_nested.yaml",
    "withparam_global.yaml",
    "withparam_global_dict.yaml",
    "withparam_output.yaml",
    "withparam_output_dict.yaml",

    # TODO: remove the following from ignored list
    # "any_sequencer.yaml",       # takes 5 min
    # "basic_no_decorator.yaml",  # takes 2 min
    # "compose.yaml",             # takes 2 min
]

if ignored_yaml_files:
    logging.warning("Ignoring the following pipelines: {}".format(
        ", ".join(ignored_yaml_files)))

# run pipelines in "kubeflow" namespace as some E2E tests depend on Minio
# for artifact storage in order to access secrets:
namespace = "kubeflow"

# experiment name to group the pipeline runs started by these E2E tests
experiment_name = "E2E_TEST"

# KFP doesn't allow any resource to be created by a pipeline. The API has an option
# for users to provide their own service account that has those permissions.
# see https://github.com/kubeflow/kfp-tekton/blob/master/sdk/sa-and-rbac.md
# TODO: add to setUpClass method
rbac = textwrap.dedent("""\
    apiVersion: rbac.authorization.k8s.io/v1beta1
    kind: ClusterRoleBinding
    metadata:
      name: default-admin
    subjects:
      - kind: ServiceAccount
        name: default
        namespace: {}
    roleRef:
      kind: ClusterRole
      name: cluster-admin
      apiGroup: rbac.authorization.k8s.io
    """.format(namespace))

# TODO: enable feature flag for custom tasks to enable E2E tests for loops
#   $ kubectl get configmap feature-flags -n tekton-pipelines -o jsonpath='{.data.enable-custom-tasks}'
#   false
#   $ kubectl patch configmap feature-flags -n tekton-pipelines -p '{"data": {"enable-custom-tasks": "true"}}'
#   configmap/feature-flags patched
#
# test_loop_over_lightweight_output
# test_loop_static
# test_parallel_join_with_argo_vars
# test_parallelfor_item_argument_resolving
# test_withitem_nested
# test_withparam_global
# test_withparam_global_dict
# test_withparam_output
# test_withparam_output_dict


# =============================================================================
#  ensure we have what we need, abort early instead of failing every test
# =============================================================================

# tests require Tekton Pipelines and Kubeflow Pipelines deployed on Kubernetes
def _verify_tekton_cluster():

    def exit_on_error(cmd, expected_output=None):
        process = run(cmd.split(), capture_output=True, timeout=10, check=False)
        if not process.returncode == 0:
            logging.error("Process returned non-zero exit code: `{}` --> `{}`"
                          .format(cmd, process.stderr.decode().strip()))
            exit(process.returncode)
        cmd_output = process.stdout.decode("utf-8").strip("'")
        if expected_output and expected_output not in cmd_output:
            logging.error("Command '{}' did not return expected output '{}': {}"
                          .format(cmd, expected_output, process.stdout))
            exit(1)
        return cmd_output

    exit_on_error("kubectl get svc tekton-pipelines-controller -n tekton-pipelines")
    exit_on_error("kubectl get svc ml-pipeline -n {}".format(namespace))
    exit_on_error("kubectl get configmap kfp-tekton-config -n {}".format(namespace))
    tkn_ver_out = exit_on_error("tkn version")
    tkn_pipeline_ver = re.search(r"^Pipeline version: (.*)$", tkn_ver_out, re.MULTILINE).group(1)
    tkn_client_ver = re.search(r"^Client version: (.*)$", tkn_ver_out, re.MULTILINE).group(1)
    assert version.parse(TKN_PIPELINE_MIN_VERSION) <= version.parse(tkn_pipeline_ver),\
        "Tekton Pipeline version must be >= {}, found '{}'".format(TKN_PIPELINE_MIN_VERSION, tkn_pipeline_ver)
    assert version.parse(TKN_CLIENT_MIN_VERSION) <= version.parse(tkn_client_ver),\
        "Tekton CLI version must be >= {}, found '{}'".format(TKN_CLIENT_MIN_VERSION, tkn_client_ver)


# verify we have a working Tekton cluster
_verify_tekton_cluster()


# =============================================================================
#  TestCase class with utility methods, the actual test_... case methods will
#  get generated dynamically below
# =============================================================================

class TestCompilerE2E(unittest.TestCase):
    """Dynamically generated end-to-end test cases taking each of the pipelines
    which were generated by the `kfp_tekton.compiler` and running them on a
    Kubernetes cluster with Tekton Pipelines installed."""

    client = TektonClient()
    verbosity = 2
    pr_name_map = dict()
    failed_tests = set()

    @classmethod
    def setUpClass(cls):
        # TODO: set up RBAC and other pre test requirements
        cls._delete_all_pipelineruns()

    @classmethod
    def tearDownClass(cls):
        # TODO: cleanup cluster resources, pods, maybe inject labels to each
        #   resource before deploying pipelines to identify the resources
        #   created during test execution and delete via label selectors after
        logging.warning("The following pipelines were ignored: {}".format(
            ", ".join(ignored_yaml_files)))
        if cls.failed_tests:
            with open(failed_tests_file, 'w') as f:
                f.write("\n".join(sorted(cls.failed_tests)))
        else:
            if os.path.exists(failed_tests_file):
                os.remove(failed_tests_file)

    def tearDown(self) -> None:
        if hasattr(self, '_outcome'):
            result = self.defaultTestResult()
            self._feedErrorsToResult(result, self._outcome.errors)
            if result.failures or result.errors:
                self.failed_tests.add(self._testMethodName.split(".")[0])

    @classmethod
    def _delete_all_pipelineruns(cls):
        logging.warning("Deleting previous '{}' pipeline runs".format(experiment_name))
        try:
            experiment = cls.client.get_experiment(experiment_name=experiment_name)
            e2e_test_runs = cls.client.list_runs(experiment_id=experiment.id, page_size=100)
            for r in e2e_test_runs.runs:
                cls.client._run_api.delete_run(id=r.id)
                # del_cmd = "tkn pipelinerun delete -f {} -n {}".format(name, namespace)
                # run(del_cmd.split(), capture_output=True, timeout=10, check=False)
        except ValueError as e:
            logging.warning(str(e))

    def _delete_pipelinerun(self, name):
        pr_name = self.pr_name_map[name]
        del_cmd = "tkn pipelinerun delete -f {} -n {}".format(pr_name, namespace)
        run(del_cmd.split(), capture_output=True, timeout=10, check=False)
        # TODO: find a better way than to sleep, but some PipelineRuns cannot
        #   be recreated right after the previous pipelineRun has been deleted
        sleep(SLEEP_BETWEEN_TEST_PHASES)

    def _start_pipelinerun(self, name, yaml_file):
        kfp_cmd = "kfp run submit -f {} -n {} -e {} -r {}".format(
            yaml_file, namespace, experiment_name, name)
        kfp_proc = run(kfp_cmd.split(), capture_output=True, timeout=10, check=False)
        self.assertEqual(kfp_proc.returncode, 0,
                         "Process returned non-zero exit code: {} -> {}".format(
                             kfp_cmd, kfp_proc.stderr))
        run_id = kfp_proc.stdout.decode().split()[1]
        wf_manifest = self.client.get_run(run_id).pipeline_runtime.workflow_manifest
        pr_name = json.loads(wf_manifest)['metadata']['name']
        self.pr_name_map[name] = pr_name
        # TODO: find a better way than to sleep, but some PipelineRuns take longer
        #   to be created and logs may not be available yet even with --follow or
        #   when attempting (and retrying) to get the pipelinerun status
        sleep(SLEEP_BETWEEN_TEST_PHASES)

    def _get_pipelinerun_status(self, name, retries: int = 60) -> str:
        pr_name = self.pr_name_map[name]
        tkn_status_cmd = "tkn pipelinerun describe %s -n %s -o jsonpath=" \
                         "'{.status.conditions[0].reason}'" % (pr_name, namespace)
        status = "Unknown"
        for i in range(0, retries):
            try:
                tkn_status_proc = run(tkn_status_cmd.split(), capture_output=True,
                                      timeout=10, check=False)
                if tkn_status_proc.returncode == 0:
                    status = tkn_status_proc.stdout.decode("utf-8").strip("'")
                    if status in ["Succeeded", "Completed", "Failed", "CouldntGetTask"]:
                        return status
                    logging.debug("tkn pipelinerun '{}' status: {} ({}/{})".format(
                        pr_name, status, i + 1, retries))
                else:
                    logging.error("Could not get pipelinerun status ({}/{}): {}".format(
                        i + 1, retries, tkn_status_proc.stderr.decode("utf-8")))
            except SubprocessError:
                logging.exception("Error trying to get pipelinerun status ({}/{})".format(
                        i + 1, retries))
            sleep(SLEEP_BETWEEN_TEST_PHASES)
        return status

    def _get_pipelinerun_logs(self, name, timeout: int = 120) -> str:
        sleep(SLEEP_BETWEEN_TEST_PHASES * 2)  # if we don't wait, we often only get logs of some pipeline tasks
        pr_name = self.pr_name_map[name]
        tkn_logs_cmd = "tkn pipelinerun logs {} -n {}".format(pr_name, namespace)
        tkn_logs_proc = run(tkn_logs_cmd.split(), capture_output=True, timeout=timeout, check=False)
        self.assertEqual(tkn_logs_proc.returncode, 0,
                         "Process returned non-zero exit code: {} -> {}".format(
                             tkn_logs_cmd, tkn_logs_proc.stderr))
        return tkn_logs_proc.stdout.decode("utf-8")

    def _verify_logs(self, name, golden_log_file, test_log):
        if GENERATE_GOLDEN_E2E_LOGS:
            with open(golden_log_file, 'w') as f:
                f.write(test_log.replace(self.pr_name_map[name], name))
        else:
            try:
                with open(golden_log_file, 'r') as f:
                    golden_log = f.read()
                sanitized_golden_log = self._sanitize_log(name, golden_log)
                sanitized_test_log = self._sanitize_log(name, test_log)
                self.maxDiff = None
                self.assertEqual(sanitized_golden_log,
                                 sanitized_test_log,
                                 msg="PipelineRun '{}' did not produce the expected "
                                     " log output: {}".format(name, golden_log_file))
            except FileNotFoundError:
                logging.error("Could not find golden log file '{}'."
                              " Generate it by re-running this test with"
                              " GENERATE_GOLDEN_E2E_LOGS='True'".format(golden_log_file))
                raise

    def _sanitize_log(self, name, log) -> str:
        """Sanitize log output by removing or replacing elements that differ
        from one pipeline execution to another:
          - timestamps like 2020-06-08T21:58:06Z, months, weekdays
          - identifiers generated by Kubernetes i.e. for pod names
          - any numbers
          - strip trailing spaces and remove empty lines

        :param log: the pipeline execution log output
        :return: the sanitized log output fit for comparing to previous logs
        """
        # copied from datetime, cannot be imported, sadly
        _DAYNAMES = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
        _MONTHNAMES = ["Jan", "Feb", "Mar", "Apr", "May", "Jun",
                       "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]

        # depending on cluster capacity and utilization and the timing of the
        # log listener some task logs contain lines that differ from test run
        # to test run, like the progress output of file copy operations or a
        # server process receiving a termination signal
        lines_to_remove = [
            "still running",
            "Server is listening on",
            "Unknown signal terminated",
            "Exiting...",
            r"Total: .+, Transferred: .+, Speed: .+",
            r"localhost:.*GET / HTTP",
        ]

        # replacements are used on multi-line strings, so '...\n' will be matched by '...$'
        replacements = [
            (self.pr_name_map[name], name),
            (r"(-[-0-9a-z]{3}-[-0-9a-z]{5})(?=[ -/\]\"]|$)", r"-XXX-XXXXX"),
            (r"[0-9a-z]{8}(-[0-9a-z]{4}){3}-[0-9a-z]{12}",
             "{}-{}-{}-{}-{}".format("X" * 8, "X" * 4, "X" * 4, "X" * 4, "X" * 12)),
            (r"resourceVersion:[0-9]+ ", "resourceVersion:-------- "),
            (r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", "DATETIME"),
            (r"{}".format("|".join(_MONTHNAMES)), "MONTH"),
            (r"{}".format("|".join(_DAYNAMES)), "DAY"),
            (r"\d+", "-"),
            (r" +$", ""),
            (r" +\r", r"\n"),
            (r"^$\n", ""),
            (r"\n^$", ""),
        ]

        sanitized_log = log

        # replace "random" generated parts of the log text
        for pattern, repl in replacements:
            sanitized_log = re.sub(pattern, repl, sanitized_log, flags=re.MULTILINE)

        # sort lines since parallel tasks produce log lined in unpredictable order
        # remove erratic lines which only show up some times
        sanitized_log = "\n".join(
            sorted(filter(lambda l: not any(re.findall(r, l) for r in lines_to_remove),
                          sanitized_log.splitlines())))

        return sanitized_log

    def _run_test__validate_tekton_yaml(self, name, yaml_file):
        if not USE_LOGS_FROM_PREVIOUS_RUN:
            self._start_pipelinerun(name, yaml_file)

    def _run_test__verify_pipelinerun_success(self, name):
        status = self._get_pipelinerun_status(name)
        self.assertIn(status, ["Succeeded", "Completed"])

    def _run_test__verify_pipelinerun_logs(self, name, log_file):
        test_log = self._get_pipelinerun_logs(name)
        self._verify_logs(name, log_file, test_log)

    # deprecated, use `self._run_test__xyz` methods separately
    def _run_test(self, name, yaml_file, log_file):
        self._run_test__validate_tekton_yaml(name, yaml_file)
        self._run_test__verify_pipelinerun_success(name)
        self._run_test__verify_pipelinerun_logs(name, log_file)


# =============================================================================
#  dynamically generate test cases from Tekton YAML files in compiler testdata
# =============================================================================

def _skip_test_and_why(yaml_file_path):

    yaml_file_name = os.path.basename(yaml_file_path)
    test_name = 'test_{0}'.format(os.path.splitext(yaml_file_name)[0])

    is_ignored = yaml_file_name in ignored_yaml_files
    is_excluded = test_name in EXCLUDE_TESTS
    not_included = INCLUDE_TESTS and test_name not in INCLUDE_TESTS
    not_was_failed_if_rerun_failed_only = RERUN_FAILED_TESTS_ONLY \
        and test_name not in previously_failed_tests

    if not_was_failed_if_rerun_failed_only:
        return (True, f"{test_name} NOT in 'previously_failed_tests'")

    if not_included:
        return (True, f"{test_name} not in 'INCLUDE_TESTS'")

    if is_excluded:
        return (True, f"{test_name} in 'EXCLUDE_TESTS'")

    if is_ignored:
        return (True, f"{yaml_file_name} in 'ignored_yaml_files'")

    return (False, "not skipped")


def _generate_test_cases(pipeline_runs: [dict]):

    def create_test_function__validate_yaml(test_name, yaml_file):
        @unittest.skipIf(*_skip_test_and_why(yaml_file))
        def test_function(self):
            self._run_test__validate_tekton_yaml(test_name, yaml_file)
        return test_function

    def create_test_function__check_run_status(test_name, yaml_file):
        @unittest.skipIf(*_skip_test_and_why(yaml_file))
        def test_function(self):
            self._run_test__verify_pipelinerun_success(test_name)
        return test_function

    def create_test_function__verify_logs(test_name, yaml_file, log_file):
        @unittest.skipIf(*_skip_test_and_why(yaml_file))
        def test_function(self):
            self._run_test__verify_pipelinerun_logs(test_name, log_file)
        return test_function

    for p in pipeline_runs:
        yaml_file_name = os.path.splitext(os.path.basename(p["yaml_file"]))[0]

        # 1. validate Tekton YAML (and kick of pipelineRun)
        setattr(TestCompilerE2E,
                'test_{0}.i_validate_yaml'.format(yaml_file_name),
                create_test_function__validate_yaml(p["name"], p["yaml_file"]))

        # 2. check pipelineRun status
        setattr(TestCompilerE2E,
                'test_{0}.ii_check_run_success'.format(yaml_file_name),
                create_test_function__check_run_status(p["name"], p["yaml_file"]))

        # 3. verify pipelineRun log output
        setattr(TestCompilerE2E,
                'test_{0}.iii_verify_logs'.format(yaml_file_name),
                create_test_function__verify_logs(p["name"], p["yaml_file"], p["log_file"]))


def _generate_test_list(file_name_expr="*.yaml") -> [dict]:

    testdata_dir = os.path.join(os.path.dirname(__file__), "testdata")
    yaml_files = sorted(glob(os.path.join(testdata_dir, file_name_expr)))
    pipeline_runs = []

    for yaml_file in yaml_files:
        with open(yaml_file, 'r') as f:
            pipeline_run = yaml.safe_load(f)
        if pipeline_run.get("kind") == "PipelineRun":
            pipeline_runs.append({
                "name": pipeline_run["metadata"]["name"],
                "yaml_file": yaml_file,
                "log_file": yaml_file.replace(".yaml", ".log")
            })
    return pipeline_runs


_generate_test_cases(_generate_test_list("*.yaml"))


# =============================================================================
#  run the E2E tests as a Python script:
#      python3 compiler/compiler_tests_e2e.py
#
#  ... as opposed to:
#      python3 -m unittest compiler.compiler_tests_e2e
#
#  ... because the test_xyz methods are dynamically generated
# =============================================================================

if __name__ == '__main__':
    unittest.main(verbosity=TestCompilerE2E.verbosity)
    logging.warning("The following pipelines were ignored: {}".format(
        ", ".join(ignored_yaml_files)))
