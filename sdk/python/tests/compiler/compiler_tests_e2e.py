# Copyright 2020 kubeflow.org
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

import logging
import os
import re
import textwrap
import unittest
import yaml

from glob import glob
from os import environ as env
from subprocess import run, SubprocessError
from time import sleep


# Temporarily set GENERATE_GOLDEN_E2E_LOGS=True to (re)generate new "golden" log
# files after making code modifications that change the expected log output.
# To (re)generate all "golden" log output files from the command line run:
#    GENERATE_GOLDEN_E2E_LOGS=True sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test GENERATE_GOLDEN_E2E_LOGS=True
GENERATE_GOLDEN_E2E_LOGS = env.get("GENERATE_GOLDEN_E2E_LOGS", "False") == "True"

# when USE_LOGS_FROM_PREVIOUS_RUN=True, the logs from the previous pipeline run
# will be use for log verification (or regenerating "golden" log files):
#    USE_LOGS_FROM_PREVIOUS_RUN=True sdk/python/tests/run_e2e_tests.sh
# or:
#    make e2e_test USE_LOGS_FROM_PREVIOUS_RUN=True
USE_LOGS_FROM_PREVIOUS_RUN = env.get("USE_LOGS_FROM_PREVIOUS_RUN", "False") == "True"

# get the Kubernetes context from the KUBECONFIG env var
KUBECONFIG = env.get("KUBECONFIG")

# ignore pipelines with unpredictable log output or complex prerequisites
# TODO: revisit this list, try to rewrite those Python DSLs in a way that they
#  will produce logs which can be verified. One option would be to keep more
#  than one "golden" log file to match either of the desired possible outputs
ignored_yaml_files = [
    "big_data_passing.yaml",    # does not complete in a reasonable time frame
    "katib.yaml",               # service account needs Katib permission, takes too long doing 9 trail runs
    "timeout.yaml",             # random failure (by design) ... would need multiple log files to compare to
    "tolerations.yaml",         # designed to fail, test show only how to add the toleration to the pod
    "volume.yaml",              # need to rework the credentials part
    "volume_snapshot_op.yaml",  # only works on Minikube, K8s alpha feature, requires a feature gate from K8s master
]

# run pipelines in "kubeflow" namespace as some E2E tests depend on Minio
# for artifact storage in order to access secrets:
namespace = "kubeflow"

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


# =============================================================================
#  make sure we have what we need, abort early instead of failing every test
# =============================================================================

# tests require Tekton Pipelines and Kubeflow Pipelines deployed on Kubernetes
def _verify_tekton_cluster():
    def exit_on_error(cmd):
        process = run(cmd.split(), capture_output=True, timeout=10, check=False)
        if not process.returncode == 0:
            logging.error("Process returned non-zero exit code: %s" % process.stderr)
            exit(process.returncode)
    exit_on_error("kubectl get svc tekton-pipelines-controller -n tekton-pipelines")
    exit_on_error("kubectl get svc ml-pipeline -n {}".format(namespace))
    exit_on_error("tkn version")


# verify we have a working Tekton cluster
if not KUBECONFIG:
    logging.error("The environment variable 'KUBECONFIG' was not set.")
    exit(1)
else:
    _verify_tekton_cluster()

# let the user know this test run is not performing any verification
if GENERATE_GOLDEN_E2E_LOGS:
    logging.warning(
        "The environment variable 'GENERATE_GOLDEN_E2E_LOGS' was set to 'True'. "
        "Test cases will (re)generate the 'golden' log files instead of verifying "
        "the logs produced by running the Tekton YAML on a Kubernetes cluster.")

# let the user know we are using the logs produced from a prior run
if USE_LOGS_FROM_PREVIOUS_RUN:
    logging.warning(
        "The environment variable 'USE_LOGS_FROM_PREVIOUS_RUN' was set to 'True'. "
        "Test cases will use the logs produced by a prior pipeline execution. "
        "The Tekton YAML will not be verified on a Kubernetes cluster.")


# =============================================================================
#  TestCase class with utility methods, the actual test_... case methods will
#  get generated dynamically below
# =============================================================================

class TestCompilerE2E(unittest.TestCase):
    """Dynamically generated end-to-end test cases taking each of the pipelines
    which were generated by the `kfp_tekton.compiler` and running them on a
    Kubernetes cluster with Tekton Pipelines installed."""

    verbosity = 2

    @classmethod
    def setUpClass(cls):
        # TODO: set up RBAC and other pre test requirements
        logging.warning("Ignoring the following pipelines: {}".format(
            ", ".join(ignored_yaml_files)))

    @classmethod
    def tearDownClass(cls):
        # TODO: cleanup cluster resources, pods, maybe inject labels to each
        #   resource before deploying pipelines to identify the resources
        #   created during test execution and delete via label selectors after
        logging.warning("The following pipelines were ignored: {}".format(
            ", ".join(ignored_yaml_files)))

    def _delete_pipelinerun(self, name):
        del_cmd = "tkn pipelinerun delete -f {} -n {}".format(name, namespace)
        run(del_cmd.split(), capture_output=True, timeout=10, check=False)
        # TODO: find a better way than to sleep, but some PipelineRuns cannot
        #   be recreated right after the previous pipelineRun has been deleted
        sleep(5)

    def _start_pipelinerun(self, yaml_file):
        kube_cmd = "kubectl apply -f \"{}\" -n {}".format(yaml_file, namespace)
        kube_proc = run(kube_cmd.split(), capture_output=True, timeout=10, check=False)
        self.assertEqual(kube_proc.returncode, 0,
                         "Process returned non-zero exit code: {} -> {}".format(
                             kube_cmd, kube_proc.stderr))
        # TODO: find a better way than to sleep, but some PipelineRuns take longer
        #   to be created and logs may not be available yet even with --follow
        sleep(5)

    def _get_pipelinerun_status(self, name, retries: int = 10) -> str:
        tkn_status_cmd = "tkn pipelinerun describe %s -n %s -o jsonpath=" \
                         "'{.status.conditions[0].type}'" % (name, namespace)
        status = "Unknown"
        for i in range(0, retries):
            try:
                tkn_status_proc = run(tkn_status_cmd.split(), capture_output=True,
                                      timeout=10, check=False)
                if tkn_status_proc.returncode == 0:
                    status = tkn_status_proc.stdout.decode("utf-8").strip("'")
                    if "Succeeded" in status or "Failed" in status:
                        return status
                    logging.warning("tkn pipeline '{}' {} ({}/{})".format(
                        name, status, i + 1, retries))
                else:
                    logging.error("Could not get pipelinerun status ({}/{}): {}".format(
                        i + 1, retries, tkn_status_proc.stderr.decode("utf-8")))
            except SubprocessError:
                logging.exception("Error trying to get pipelinerun status ({}/{})".format(
                        i + 1, retries))
            sleep(3)
        return status

    def _get_pipelinerun_logs(self, name, timeout: int = 30) -> str:
        sleep(10)  # if we don't wait, we often only get logs of some pipeline tasks
        tkn_logs_cmd = "tkn pipelinerun logs {} -n {}".format(name, namespace)
        tkn_logs_proc = run(tkn_logs_cmd.split(), capture_output=True, timeout=timeout, check=False)
        self.assertEqual(tkn_logs_proc.returncode, 0,
                         "Process returned non-zero exit code: {} -> {}".format(
                             tkn_logs_cmd, tkn_logs_proc.stderr))
        return tkn_logs_proc.stdout.decode("utf-8")

    def _verify_logs(self, name, golden_log_file, test_log):
        if GENERATE_GOLDEN_E2E_LOGS:
            with open(golden_log_file, 'w') as f:
                f.write(test_log)
        else:
            try:
                with open(golden_log_file, 'r') as f:
                    golden_log = f.read()
                self.maxDiff = None
                self.assertEqual(self._sanitize_log(golden_log),
                                 self._sanitize_log(test_log),
                                 msg="PipelineRun '{}' did not produce the expected "
                                     " log output: {}".format(name, golden_log_file))
            except FileNotFoundError:
                logging.error("Could not find golden log file '{}'."
                              " Generate it by re-running this test with"
                              " GENERATE_GOLDEN_E2E_LOGS='True'".format(golden_log_file))
                raise

    def _sanitize_log(self, log) -> str:
        """Sanitize log output by removing or replacing elements that differ
        from one pipeline execution to another:

          - timestamps like 2020-06-08T21:58:06Z, months, weekdays
          - identifiers generated by Kubernetes i.e. for pod names
          - any numbers
          - strip empty lines

        :param log: the pipeline execution log output
        :return: the sanitized log output fit for comparing to previous logs
        """
        # copied from datetime, cannot be imported, sadly
        _DAYNAMES = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
        _MONTHNAMES = ["Jan", "Feb", "Mar", "Apr", "May", "Jun",
                       "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]

        replacements = [
            ("Pipeline still running ...\n", ""),
            (r"(-[-0-9a-z]{3}-[-0-9a-z]{5})(?=[ -/\]\"]|$)", r"-XXX-XXXXX"),
            (r"uid:[0-9a-z]{8}(-[0-9a-z]{4}){3}-[0-9a-z]{12}",
             "uid:{}-{}-{}-{}-{}".format("X" * 8, "X" * 4, "X" * 4, "X" * 4, "X" * 12)),
            (r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", "DATETIME"),
            (r"{}".format("|".join(_MONTHNAMES)), "MONTH"),
            (r"{}".format("|".join(_DAYNAMES)), "DAY"),
            (r"\d", "-"),
            (r"^$\n", ""),
            (r"\n^$", ""),
        ]
        sanitized_log = log

        # replace "random" generated parts of the log text
        for pattern, repl in replacements:
            sanitized_log = re.sub(pattern, repl, sanitized_log, flags=re.MULTILINE)

        # sort lines since parallel tasks produce log lined in unpredictable order
        sanitized_log = "\n".join(sorted(sanitized_log.splitlines()))

        return sanitized_log

    def _run_test__validate_tekton_yaml(self, name, yaml_file):
        if not USE_LOGS_FROM_PREVIOUS_RUN:
            self._delete_pipelinerun(name)
            self._start_pipelinerun(yaml_file)

    def _run_test__verify_pipelinerun_success(self, name):
        status = self._get_pipelinerun_status(name)
        self.assertEqual("Succeeded", status)

    def _run_test__verify_pipelinerun_logs(self, name, log_file):
        test_log = self._get_pipelinerun_logs(name)
        self._verify_logs(name, log_file, test_log)

    def _run_test(self, name, yaml_file, log_file):
        self._run_test__validate_tekton_yaml(name, yaml_file)
        self._run_test__verify_pipelinerun_success(name)
        self._run_test__verify_pipelinerun_logs(name, log_file)


# =============================================================================
#  dynamically generate test cases from Tekton YAML files in compiler testdata
# =============================================================================

def _generate_test_cases(pipeline_runs: [dict]):

    def create_test_function__validate_yaml(test_name, yaml_file):
        def test_function(self):
            self._run_test__validate_tekton_yaml(test_name, yaml_file)
        return test_function

    def create_test_function__check_run_status(test_name):
        def test_function(self):
            self._run_test__verify_pipelinerun_success(test_name)
        return test_function

    def create_test_function__verify_logs(test_name, log_file):
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
                create_test_function__check_run_status(p["name"]))

        # 3. verify pipelineRun log output
        setattr(TestCompilerE2E,
                'test_{0}.iii_verify_logs'.format(yaml_file_name),
                create_test_function__verify_logs(p["name"], p["log_file"]))


def _generate_test_list(file_name_expr="*.yaml") -> [dict]:
    def is_not_ignored(yaml_file):
        return not any(name_part in yaml_file
                       for name_part in ignored_yaml_files)

    testdata_dir = os.path.join(os.path.dirname(__file__), "testdata")
    yaml_files = filter(is_not_ignored, glob(os.path.join(testdata_dir, file_name_expr)))
    pipeline_runs = []
    for yaml_file in yaml_files:
        with open(yaml_file, 'r') as f:
            pipeline_run = yaml.safe_load(f)
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
