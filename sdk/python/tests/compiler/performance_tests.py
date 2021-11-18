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

from datetime import datetime as dt
from os import environ as env
from typing import Mapping, Callable

from kfp_server_api import ApiRun, ApiRunDetail, ApiException
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
OUTPUT_FILE = env.get("OUTPUT_FILE", f"perf_test_{dt.now().strftime('%Y%m%d_%H%M%S')}_{PUBLIC_IP}.csv")
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
#  local settings that are not loaded from env variables
# =============================================================================

# kfp_tekton_root_dir = os.path.abspath(__file__).replace("sdk/python/tests/compiler/performance_tests.py", "")


def get_client() -> TektonClient:
    host = f"http://{PUBLIC_IP}/pipeline"
    cookies = f"connect.sid={CONNECT_SID}; userinfo={USER_INFO}" if CONNECT_SID and USER_INFO else None
    client = TektonClient(host=host, cookies=cookies)
    client.set_user_namespace(NAMESPACE)  # overwrite system default with None if necessary
    return client


# method annotation to ensure the wrapped function is executed synchronously
def synchronized(function):
    lock = threading.Lock()

    @functools.wraps(function)
    def _synchronized_function(*args, **kwargs):
        with lock:
            result = function(*args, **kwargs)
            return result

    return _synchronized_function


# TODO: synchronizing pipeline compilation skews recorded compile times since some
#   pipelines will wait for others to be compiled, yet the recorded compilation
#   start times are equal for all pipelines
#   We need to change test design to run and time pipeline compilation sequentially
#   and only execute pipelines in parallel
if NUM_WORKERS > 1:
    print("WARNING: pipeline compilation times are not accurate when running in parallel.\n")


# TODO: cannot compile multiple pipelines in parallel due to use of static variables
#   causing Exception "Nested pipelines are not allowed." in kfp/dsl/_pipeline.py
#   def __enter__(self):
#     if Pipeline._default_pipeline:
#       raise Exception('Nested pipelines are not allowed.')
@synchronized
def compile_pipeline(pipeline_func: Callable) -> str:
    pipeline_name = pipeline_func.__name__
    file_name = pipeline_name + '.yaml'
    tmpdir = tempfile.gettempdir()  # TODO: keep compiled pipelines?
    pipeline_package_path = os.path.join(tmpdir, file_name)
    pipeline_conf = TektonPipelineConf()
    TektonCompiler().compile(pipeline_func=pipeline_func,
                             package_path=pipeline_package_path,
                             pipeline_conf=pipeline_conf)
    return pipeline_package_path


def run_pipeline(pipeline_file: str,
                 run_name: str,
                 arguments: Mapping[str, str] = None):

    client = get_client()
    experiment = client.create_experiment(EXPERIMENT)  # get or create
    run_result = None
    while run_result is None:  # TODO: add timeout or max retries on ApiException
        try:
            run_result: ApiRun = client.run_pipeline(
                experiment_id=experiment.id,
                job_name=run_name,
                pipeline_package_path=pipeline_file,
                params=arguments)
        except ApiException as e:
            print(f"KFP Server Exception: '{e.reason}' {e.status} '{e.body}'"
                  f" {datetime.datetime.now().strftime('%Y/%m/%d %H:%M:%S')}")
            time.sleep(1)
            client = get_client()

    return run_result.id


def wait_for_run_to_complete(run_id: str) -> ApiRun:
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
    print(f"{run.name.ljust(20)[:20]}"
          f" {run.status.lower().ljust(10)[:10]}"
          f" after {(run.finished_at - run.created_at)}"
          f" ({run.created_at.strftime('%H:%M:%S')}->{run.finished_at.strftime('%H:%M:%S')})")
    return run


def load_pipeline_functions() -> [(Callable, str)]:
    pipeline_functions = []

    from testdata.sequential import sequential_pipeline
    pipeline_functions.append((sequential_pipeline, "sequential_pipeline"))

    from testdata.condition import flipcoin
    pipeline_functions.append((flipcoin, "flipcoin"))

    from testdata.compose import save_most_frequent_word
    pipeline_functions.append((save_most_frequent_word, "compose"))

    from testdata.retry import retry_sample_pipeline
    pipeline_functions.append((retry_sample_pipeline, "retry"))

    from testdata.loop_static import pipeline as loop_static
    pipeline_functions.append((loop_static, "loop_static"))

    from testdata.conditions_and_loops import conditions_and_loops
    pipeline_functions.append((conditions_and_loops, "conditions_and_loops"))

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


def run_single_pipeline_performance_test(pipeline_func: Callable, run_name: str) -> ApiRun:
    t = [dt.now()]

    pipeline_file = compile_pipeline(pipeline_func)
    t += [dt.now()]

    run_id = run_pipeline(pipeline_file, run_name)
    t += [dt.now()]

    run_details = wait_for_run_to_complete(run_id)  # noqa F841
    t += [dt.now()]

    with open(OUTPUT_FILE, "a") as f:
        time_deltas = [str(t[i + 1] - t[i]) for i in range(len(t) - 1)]
        f.write(OUTPUT_SEP.join([run_name] + time_deltas) + "\n")

    return run_details


def run_concurrently(pipelinefunc_name_tuples: [(Callable, str)]) -> [(str, str)]:
    pipeline_status = []

    with concurrent.futures.ThreadPoolExecutor(max_workers=NUM_WORKERS) as executor:
        performance_tests = (
            executor.submit(run_single_pipeline_performance_test, func, name)
            for (func, name) in pipelinefunc_name_tuples
        )
        for performance_test in concurrent.futures.as_completed(performance_tests):
            try:
                run_details = performance_test.result()
                pipeline_status.append((run_details.name, run_details.status))
            except Exception as e:
                error = f"{e.__class__.__name__}: {str(e)}"
                print(error)
                pipeline_status.append(("unknown pipeline", error))

    return pipeline_status


def run_performance_tests():

    with open(OUTPUT_FILE, "w") as f:
        f.write(OUTPUT_SEP.join(["Pipeline", "Compile", "Submit", "Run"]) + "\n")

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
