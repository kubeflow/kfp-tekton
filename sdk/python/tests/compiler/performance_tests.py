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
import sys  # noqa
import tempfile
import time
import threading
import json

from collections import defaultdict
from datetime import datetime as dt
from datetime import timedelta
from os import environ as env
from typing import Callable, Dict, Mapping

from kfp_server_api import ApiException, ApiRun, ApiRunDetail
from kfp_tekton.compiler import TektonCompiler
from kfp_tekton._client import TektonClient
from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf


# =============================================================================
#  load test settings from environment variables
# =============================================================================

# TODO: turn env vars into script parameters
PUBLIC_IP = env.get("PUBLIC_IP")
NAMESPACE = env.get("NAMESPACE", None)
USER_INFO = env.get("USER_INFO")
CONNECT_SID = env.get("CONNECT_SID")
NUM_WORKERS = int(env.get("NUM_WORKERS", 1))
EXPERIMENT = env.get("EXPERIMENT_NAME", "PERF_TEST")
OUTPUT_FILE = env.get("OUTPUT_FILE", f"perf_test_{dt.now().strftime('%Y%m%d_%H%M%S')}_N{NUM_WORKERS}_{PUBLIC_IP}.csv")
OUTPUT_SEP = env.get("OUTPUT_SEP", ",")


print(f"Environment variables:\n\n"
      f"  PUBLIC_IP:   {PUBLIC_IP}\n"
      f"  NAMESPACE:   {NAMESPACE}\n"
      f"  USER_INFO:   {USER_INFO}\n"
      f"  CONNECT_SID: {CONNECT_SID}\n"
      f"  NUM_WORKERS: {NUM_WORKERS}\n"
      f"  EXPERIMENT:  {EXPERIMENT}\n"
      f"  OUTPUT_FILE: {OUTPUT_FILE}\n"
      f"  OUTPUT_SEP:  {OUTPUT_SEP}\n")


# =============================================================================
#  local variables
# =============================================================================

# kfp_tekton_root_dir = os.path.abspath(__file__).replace("sdk/python/tests/compiler/performance_tests.py", "")

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
                     pipeline_func: Callable) -> str:

    file_name = pipeline_name + '.yaml'
    tmpdir = tempfile.gettempdir()  # TODO: keep compiled pipelines?
    pipeline_package_path = os.path.join(tmpdir, file_name)
    pipeline_conf = TektonPipelineConf()
    TektonCompiler().compile(pipeline_func=pipeline_func,
                             package_path=pipeline_package_path,
                             pipeline_conf=pipeline_conf)
    return pipeline_package_path


@time_it
def submit_pipeline_run(*,  # force kwargs for time_it decorator to get pipeline_name
                        pipeline_name: str,
                        pipeline_file: str,
                        arguments: Mapping[str, str] = None):

    client = get_client()
    experiment = client.create_experiment(EXPERIMENT)  # get or create
    run_result = None
    while run_result is None:  # TODO: add timeout or max retries on ApiException
        try:
            run_result: ApiRun = client.run_pipeline(
                experiment_id=experiment.id,
                job_name=pipeline_name,
                pipeline_package_path=pipeline_file,
                params=arguments)
        except ApiException as e:
            print(f"KFP Server Exception: '{e.reason}' {e.status} '{e.body}'"
                  f" {datetime.datetime.now().strftime('%Y/%m/%d %H:%M:%S')}")
            time.sleep(1)
            client = get_client()

    return run_result.id


@time_it
def wait_for_run_to_complete(*,  # force kwargs so the time_it decorator can get pipeline_name
                             pipeline_name: str,
                             run_id: str) -> ApiRunDetail:
    client = get_client()
    status = None

    while status not in ["Succeeded", "Failed", "Error", "Skipped", "Terminated",
                         "Completed", "CouldntGetTask"]:
        try:
            run_detail: ApiRunDetail = client.get_run(run_id)
            run: ApiRun = run_detail.run
            status = run.status
        except ApiException as e:  # TODO: add timeout or max retries on ApiError
            print(f"KFP Server Exception: {e.reason}")
            time.sleep(1)

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


def load_pipeline_functions() -> [(Callable, str)]:
    pipeline_functions = []

    # Sequential pipelines
    from testdata.condition_sample import flipcoin_pipeline as normal_flipcoin_pipeline
    pipeline_functions.append((normal_flipcoin_pipeline, "flipcoin_normal_pipeline"))

    from testdata.condition_custom import flipcoin_pipeline as custom_pipeline
    pipeline_functions.append((custom_pipeline, "flipcoin_custom_pipeline"))

    from testdata.trusted_ai import trusted_ai
    pipeline_functions.append((trusted_ai, "trusted_ai"))

    from testdata.calc_pipeline import calc_pipeline
    pipeline_functions.append((calc_pipeline, "python_script_pipeline"))

    from testdata.sequential import sequential_pipeline
    pipeline_functions.append((sequential_pipeline, "sequential_pipeline"))


    # Parallel pipelines
    # from testdata.withitem_nested import pipeline as nested_loop_pipeline
    # pipeline_functions.append((nested_loop_pipeline, "nested_loop_pipeline"))

    # from testdata.condition import flipcoin
    # pipeline_functions.append((flipcoin, "flipcoin"))

    # from testdata.retry import retry_sample_pipeline
    # pipeline_functions.append((retry_sample_pipeline, "retry"))

    # from testdata.loop_static import pipeline as loop_static
    # pipeline_functions.append((loop_static, "loop_static"))

    # from testdata.conditions_and_loops import conditions_and_loops
    # pipeline_functions.append((conditions_and_loops, "conditions_and_loops"))

    # from testdata.loop_in_recursion import flipcoin as loop_in_loop
    # pipeline_functions.append((loop_in_loop, "loop_in_recursion"))
    #
    # from testdata.condition_custom_task import flipcoin_pipeline
    # pipeline_functions.append((flipcoin_pipeline, "condition_custom_task"))

    # TODO: add more pipelines

    # NOTE: loading samples from outside package scope is hacky
    # sys.path.insert(1, '/Users/dummy/projects/kfp-tekton/samples/lightweight-component')
    # from calc_pipeline import calc_pipeline
    # pipeline_functions.append(calc_pipeline)

    return pipeline_functions


def run_concurrently(pipelinefunc_name_tuples: [(Callable, str)]) -> [(str, str)]:
    pipeline_status = []

    with concurrent.futures.ThreadPoolExecutor(max_workers=NUM_WORKERS) as executor:
        performance_tests = (
            executor.submit(run_single_pipeline_performance_test, func, name)
            for (func, name) in pipelinefunc_name_tuples
        )
        for performance_test in concurrent.futures.as_completed(performance_tests):
            try:
                run_details = performance_test.result().run
                pipeline_status.append((run_details.name, run_details.status))
            except Exception as e:
                error = f"{e.__class__.__name__}: {str(e)}"
                print(error)
                pipeline_status.append(("unknown pipeline", error))

    return pipeline_status


def run_single_pipeline_performance_test(pipeline_func: Callable,
                                         pipeline_name: str) -> ApiRunDetail:

    pipeline_file = compile_pipeline(pipeline_name=pipeline_name, pipeline_func=pipeline_func)
    run_id = submit_pipeline_run(pipeline_name=pipeline_name, pipeline_file=pipeline_file)
    run_details = wait_for_run_to_complete(pipeline_name=pipeline_name, run_id=run_id)
    task_details = parse_run_details(run_details)

    append_exec_times_to_output_file(pipeline_name, task_details)

    return run_details


def parse_run_details(run_details: ApiRunDetail) -> dict:
    rev = {}
    pipelinerun = json.loads(run_details.to_dict()["pipeline_runtime"]["workflow_manifest"])
    status = pipelinerun["status"]

    def get_details(data):
        info = {}
        total = timedelta(0)
        count = 0
        for key in data.keys():
            run = data[key]
            status = run["status"]
            conditions = status["conditions"]
            state = conditions[len(conditions) - 1]['type']
            elapsed = dt.strptime(status['completionTime'], "%Y-%m-%dT%H:%M:%SZ") - dt.strptime(status['startTime'], "%Y-%m-%dT%H:%M:%SZ")
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
        rev["taskRuns"] = get_details(status["taskRuns"])

    if "runs" in status:
        rev["runs"] = get_details(status["runs"])

    return rev


def append_exec_times_to_output_file(pipeline_name: str, tasks: dict):

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

    with open(OUTPUT_FILE, "a") as f:
        f.write(OUTPUT_SEP.join([
            pipeline_name, str(compile_time), str(submit_time), str(run_time),
            str(taskruns), str(runs), str(taskrun_elapsed), str(run_elapsed)
        ]))
        f.write("\n")


def create_output_file():

    with open(OUTPUT_FILE, "w") as f:
        f.write(OUTPUT_SEP.join(["Pipeline", "Compile", "Submit", "Run",
                "Num_TaskRuns", "Num_Runs", "Total_TaskRun_Time", "Total_Run_Time"]) + "\n")


def run_performance_tests():

    create_output_file()
    pipeline_functions = load_pipeline_functions()

    if NUM_WORKERS == 1:  # TODO: use `run_concurrently()` even with 1 worker
        for func, name in pipeline_functions:
            run_single_pipeline_performance_test(func, name)
    else:
        run_concurrently(pipeline_functions)


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
