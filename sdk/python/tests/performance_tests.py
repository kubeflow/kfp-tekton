#!/usr/bin/env python3

# Copyright 2021 kubeflow.org
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

import concurrent.futures
import datetime
import functools
import os
import tempfile
import time
import threading
import json
import yaml

from collections import defaultdict
from datetime import datetime as dt
from datetime import timedelta
from os import environ as env
from os.path import pathsep
from pathlib import Path
from typing import Dict, Mapping

from kfp_server_api import ApiException, ApiRun, ApiRunDetail
from kfp_tekton.compiler.main import compile_pyfile
from kfp_tekton._client import TektonClient
from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf


# =============================================================================
#  load test settings from environment variables
# =============================================================================

# TODO: turn env vars into script parameters, use argparse
PUBLIC_IP = env.get("PUBLIC_IP")
NAMESPACE = env.get("NAMESPACE", None)
USER_INFO = env.get("USER_INFO")
CONNECT_SID = env.get("CONNECT_SID")
NUM_WORKERS = int(env.get("NUM_WORKERS", 1))
TEST_CONFIG = env.get("TEST_CONFIG") or Path(__file__).parents[0].joinpath("perf_test_config.yaml")
EXPERIMENT = env.get("EXPERIMENT_NAME", "PERF_TEST")
OUTPUT_FILE = env.get("OUTPUT_FILE", f"perf_test_{dt.now().strftime('%Y%m%d_%H%M%S')}_N{NUM_WORKERS}_{PUBLIC_IP}.csv")
OUTPUT_SEP = env.get("OUTPUT_SEP", ",")


print(f"Environment variables:\n\n"
      f"  PUBLIC_IP:   {PUBLIC_IP}\n"
      f"  NAMESPACE:   {NAMESPACE}\n"
      f"  USER_INFO:   {USER_INFO}\n"
      f"  CONNECT_SID: {CONNECT_SID}\n"
      f"  NUM_WORKERS: {NUM_WORKERS}\n"
      f"  TEST_CONFIG: {TEST_CONFIG}\n"
      f"  EXPERIMENT:  {EXPERIMENT}\n"
      f"  OUTPUT_FILE: {OUTPUT_FILE}\n"
      f"  OUTPUT_SEP:  {OUTPUT_SEP}\n")


# =============================================================================
#  local variables
# =============================================================================

execution_times: Dict[str, Dict[str, timedelta]] = defaultdict(dict)


def record_execution_time(pipeline_name: str,
                          function_name: str,
                          execution_time: timedelta):

    execution_times[pipeline_name][function_name] = execution_time


# method annotation to record execution times
def time_it(function):

    @functools.wraps(function)
    def _timed_function(*args, **kwargs):

        start_time = dt.now()

        functions_result = function(*args, **kwargs)

        execution_time = dt.now() - start_time

        assert "pipeline_name" in kwargs, \
            f"The function '{function.__name__}' has to be invoked with keyword" \
            f" argument parameter 'pipeline_name'."

        record_execution_time(pipeline_name=kwargs["pipeline_name"],
                              function_name=function.__name__,
                              execution_time=execution_time)

        return functions_result

    return _timed_function


# method annotation to ensure the wrapped function is executed synchronously
def synchronized(function):
    lock = threading.Lock()

    @functools.wraps(function)
    def _synchronized_function(*args, **kwargs):
        with lock:
            result = function(*args, **kwargs)
            return result

    return _synchronized_function


# TODO: cannot compile multiple pipelines in parallel due to use of static variables
#   causing Exception "Nested pipelines are not allowed." in kfp/dsl/_pipeline.py
#   def __enter__(self):
#     if Pipeline._default_pipeline:
#       raise Exception('Nested pipelines are not allowed.')
# NOTE: synchronizing the method compile_pipeline could skews the recorded compilation
#   times since the start time for all pipelines are equal, but some pipelines will
#   wait for others to be compiled with the wait time included in the total compilation
#   time.
#   We need to change test design to run and time pipeline compilation sequentially
#   and only execute pipelines in parallel
@synchronized  # keep decorator precedence: synchronize outside of (before) time_it
@time_it       # time_it inside the synchronized block so idle wait is not recorded
def compile_pipeline(*,  # force kwargs for time_it decorator to get pipeline_name
                     pipeline_name: str,
                     pipeline_script: Path) -> str:

    file_name = pipeline_name + '.yaml'
    tmpdir = tempfile.gettempdir()  # TODO: keep compiled pipelines?
    pipeline_package_path = os.path.join(tmpdir, file_name)
    pipeline_conf = TektonPipelineConf()

    try:
        compile_pyfile(pyfile=pipeline_script,
                       function_name=None,
                       output_path=pipeline_package_path,
                       type_check=True,
                       tekton_pipeline_conf=pipeline_conf)

    except ValueError as e:
        print(f"{e.__class__.__name__} trying to compile {pipeline_script}: {str(e)}")

    # TODO: delete those files after running test or keep for inspection?
    return pipeline_package_path


@time_it
def submit_pipeline_run(*,  # force kwargs for time_it decorator to get pipeline_name
                        pipeline_name: str,
                        pipeline_file: str,
                        arguments: Mapping[str, str] = None):

    client = get_client()
    experiment = client.create_experiment(EXPERIMENT)  # get or create

    try:
        run_result: ApiRun = client.run_pipeline(
            experiment_id=experiment.id,
            job_name=pipeline_name,
            pipeline_package_path=pipeline_file,
            params=arguments)
        return run_result.id

    except ApiException as e:
        print(f"KFP Server Exception trying to submit pipeline {pipeline_file}:"
              f" '{e.reason}' {e.status} '{e.body}'"
              f" {datetime.datetime.now().strftime('%Y/%m/%d %H:%M:%S')}")

    except Exception as e:
        print(f"Exception trying to submit pipeline {pipeline_file}:"
              f" '{str(e)}'"
              f" {datetime.datetime.now().strftime('%Y/%m/%d %H:%M:%S')}")

    return None


@time_it
def wait_for_run_to_complete(*,  # force kwargs so the time_it decorator can get pipeline_name
                             pipeline_name: str,
                             run_id: str) -> ApiRunDetail:
    if not run_id:
        return None

    client = get_client()
    status = None

    while status not in ["Succeeded", "Failed", "Error", "Skipped", "Terminated",
                         "Completed", "CouldntGetTask"]:
        try:
            run_detail: ApiRunDetail = client.get_run(run_id)
            run: ApiRun = run_detail.run
            status = run.status
        except ApiException as e:  # TODO: add timeout or max retries on ApiError
            print(f"KFP Server Exception waiting for {pipeline_name} run {run_id}: {e.reason}")
            time.sleep(10)

        time.sleep(0.1)

    print(f"{pipeline_name.ljust(20)[:20]}"
          f" {run.status.lower().ljust(10)[:10]}"
          f" after {(run.finished_at - run.created_at)}"
          f" ({run.created_at.strftime('%H:%M:%S')}->{run.finished_at.strftime('%H:%M:%S')})")

    return run_detail


def get_client() -> TektonClient:

    host = f"http://{PUBLIC_IP}/pipeline"
    cookies = f"connect.sid={CONNECT_SID}; userinfo={USER_INFO}" if CONNECT_SID and USER_INFO else None
    client = TektonClient(host=host, cookies=cookies)
    client.set_user_namespace(NAMESPACE)  # overwrite system default with None if necessary

    return client


def get_project_root_dir() -> Path:

    script_path_presumed = "sdk/python/tests/performance_tests.py"
    script_path_actually = Path(__file__)
    project_root_folder = script_path_actually.parents[3]

    assert script_path_actually == project_root_folder.joinpath(script_path_presumed), \
        "Can not determine project root folder. Was this script file moved or renamed?"

    return project_root_folder


def load_test_config() -> dict:

    # script_path = Path(__file__)
    # script_dir = script_path.parents[0]
    # config_file = script_dir.joinpath("perf_test_config.yaml")

    with open(TEST_CONFIG, "r") as f:
        test_config = yaml.safe_load(f)

    return test_config


def load_pipeline_scripts() -> [(Path, str)]:

    pipeline_files_with_name = []
    test_config = load_test_config()
    project_dir = get_project_root_dir()

    for path_name_dict in test_config["pipeline_scripts"]:

        path = path_name_dict["path"]
        name = path_name_dict.get("name") or Path(path).stem
        copies = path_name_dict.get("copies", 1)
        if not path.startswith(pathsep):
            # path assumed to be relative to project root
            fp: Path = project_dir.joinpath(path)
        else:
            # path is absolute
            fp = Path(path)

        assert fp.exists(), f"Cannot find file: {fp.resolve()}"
        for i in range(int(copies)):
            pipeline_files_with_name.append((fp, f'{name}{i}'))

    print(f"Loaded {len(pipeline_files_with_name)} pipelines from {TEST_CONFIG}\n")

    return pipeline_files_with_name


def run_concurrently(pipelinescript_name_tuples: [(Path, str)]):

    with concurrent.futures.ThreadPoolExecutor(max_workers=NUM_WORKERS) as executor:
        performance_tests = (
            executor.submit(run_single_pipeline_performance_test, pipeline_script, name)
            for (pipeline_script, name) in pipelinescript_name_tuples
        )
        for performance_test in concurrent.futures.as_completed(performance_tests):
            try:
                run_details = performance_test.result()  # noqa F841
            except Exception as e:
                error = f"{e.__class__.__name__}: {str(e)}"
                print(error)


def run_single_pipeline_performance_test(pipeline_script: Path,
                                         pipeline_name: str):
    try:
        pipeline_file = compile_pipeline(pipeline_name=pipeline_name, pipeline_script=pipeline_script)
        run_id = submit_pipeline_run(pipeline_name=pipeline_name, pipeline_file=pipeline_file)
        run_details = wait_for_run_to_complete(pipeline_name=pipeline_name, run_id=run_id)

        status = run_details.run.status if run_details else "Error"
        task_details = parse_run_details(run_details)

        append_exec_times_to_output_file(pipeline_name, status, task_details)

    except Exception as e:
        error = f"{e.__class__.__name__} while testing '{pipeline_name}': {str(e)}"
        print(error)


def parse_run_details(run_details: ApiRunDetail) -> dict:
    task_details = {}

    if not run_details:
        return {}

    pipelinerun = json.loads(run_details.to_dict()["pipeline_runtime"]["workflow_manifest"])
    status = pipelinerun["status"]

    def get_details(data):

        info = {}
        total = timedelta(0)
        count = 0
        dt_fmt = "%Y-%m-%dT%H:%M:%SZ"

        for key in data.keys():
            run = data[key]
            status = run["status"]
            conditions = status["conditions"]
            state = conditions[len(conditions) - 1]['type']
            elapsed = dt.strptime(status['completionTime'], dt_fmt) - dt.strptime(status['startTime'], dt_fmt)
            info[run['pipelineTaskName']] = {
                "elapsed": elapsed,
                "status": state
            }
            count += 1
            total += elapsed

        info["count"] = count
        info["total_elapsed"] = total
        return info

    if "taskRuns" in status:
        task_details["taskRuns"] = get_details(status["taskRuns"])

    if "runs" in status:
        task_details["runs"] = get_details(status["runs"])

    return task_details


def append_exec_times_to_output_file(pipeline_name: str,
                                     status: str = "",
                                     tasks: dict = {}):

    compile_time = execution_times[pipeline_name][compile_pipeline.__name__]
    submit_time = execution_times[pipeline_name][submit_pipeline_run.__name__]
    run_time = execution_times[pipeline_name][wait_for_run_to_complete.__name__]
    taskruns = 0
    taskrun_elapsed = timedelta(0)
    runs = 0
    run_elapsed = timedelta(0)

    if "taskRuns" in tasks:
        taskruns = tasks["taskRuns"]["count"]
        taskrun_elapsed = tasks["taskRuns"]["total_elapsed"]

    if "runs" in tasks:
        runs = tasks["runs"]["count"]
        run_elapsed = tasks["runs"]["total_elapsed"]

    taskruns_average = taskrun_elapsed / taskruns if taskruns > 0 else taskrun_elapsed
    runs_average = run_elapsed / runs if runs > 0 else run_elapsed
    non_task_average = (taskrun_elapsed + run_elapsed) / (taskruns + runs) if (taskruns + runs) > 0 \
                        else (taskrun_elapsed + run_elapsed)

    with open(OUTPUT_FILE, "a") as f:
        f.write(OUTPUT_SEP.join([
            pipeline_name, status, str(compile_time), str(submit_time), str(run_time),
            str(taskruns), str(runs), str(taskrun_elapsed), str(run_elapsed),
            str(taskrun_elapsed + run_elapsed), str(run_time - (taskrun_elapsed + run_elapsed)),
            str(taskruns_average), str(runs_average),
            str(taskruns + runs), str(non_task_average)
        ]))
        f.write("\n")


def create_output_file():

    with open(OUTPUT_FILE, "w") as f:

        f.write(OUTPUT_SEP.join([
            "Pipeline", "Status", "Compile_Time", "Submit_Time", "Run_Time",
            "Num_TaskRuns", "Num_Runs", "Total_TaskRun_Time", "Total_Run_Time",
            "Total_Time_Spent_On_Tasks", "Time_Spent_Outside_Of_Tasks", "Average_Taskrun_Time",
            "Average_RunCR_Time", "Total_Number_Of_Tasks", "Average_Time_Spent_Outside_Of_Tasks"
        ]))
        f.write("\n")


def run_performance_tests():

    create_output_file()
    pipeline_scripts = load_pipeline_scripts()

    if NUM_WORKERS == 1:  # TODO: use `run_concurrently()` even with 1 worker
        for script, name in pipeline_scripts:
            run_single_pipeline_performance_test(script, name)
    else:
        run_concurrently(pipeline_scripts)


if __name__ == '__main__':

    run_performance_tests()

    # client = get_client()
    # from kfp_server_api import ApiListExperimentsResponse, ApiExperiment, ApiListRunsResponse
    # experiments: ApiListExperimentsResponse = client.list_experiments()
    # experiment: ApiExperiment = client.create_experiment(name='PERF_TESTS')
    # experiment: ApiExperiment = client.get_experiment(experiment_name='PERF_TEST')
    # runs: ApiListRunsResponse = client.list_runs(experiment_id=experiment.id, page_size=100)
    # print("Experiments: " + ", ".join([e.name for e in experiments.experiments]))
    # print("Runs: " + ", ".join([r.name for r in runs.runs]))
