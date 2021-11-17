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

import datetime
import os
# import sys
import tempfile
import time

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

# TODO: remove defaults below, add instructions and/or script parameters
PUBLIC_IP = env.get("PUBLIC_IP")
NAMESPACE = env.get("NAMESPACE", None)
USER_INFO = env.get("USER_INFO")
CONNECT_SID = env.get("CONNECT_SID")

print(f"Environment variables:\n"
      f"  PUBLIC_IP:   {PUBLIC_IP}\n"
      f"  NAMESPACE:   {NAMESPACE}\n"
      f"  USER_INFO:   {USER_INFO}\n"
      f"  CONNECT_SID: {CONNECT_SID}\n")


# =============================================================================
#  local settings that are not loaded from env variables
# =============================================================================

# kfp_tekton_root_dir = os.path.abspath(__file__).replace("sdk/python/tests/compiler/performance_tests.py", "")

experiment_name = "PERF_TEST"


def get_client() -> TektonClient:
    host = f"http://{PUBLIC_IP}/pipeline"
    cookies = f"connect.sid={CONNECT_SID}; userinfo={USER_INFO}" if CONNECT_SID and USER_INFO else None
    client = TektonClient(host=host, cookies=cookies)
    client.set_user_namespace(NAMESPACE)  # overwrite system default with None if necessary
    return client


def compile_pipeline(pipeline_func: Callable) -> str:
    pipeline_name = pipeline_func.__name__
    file_name = pipeline_name + '.yaml'
    tmpdir = tempfile.gettempdir()
    pipeline_package_path = os.path.join(tmpdir, file_name)
    pipeline_conf = TektonPipelineConf()
    TektonCompiler().compile(pipeline_func=pipeline_func,
                             package_path=pipeline_package_path,
                             pipeline_conf=pipeline_conf)
    return pipeline_package_path


def run_pipeline(pipeline_file: str,
                 arguments: Mapping[str, str] = None):
    client = get_client()
    pipeline_name = os.path.splitext(os.path.basename(pipeline_file))[0]
    run_name = pipeline_name  # + ' ' + datetime.datetime.now().strftime('%Y-%m-%d %H-%M-%S')
    experiment = client.create_experiment(experiment_name)  # get or create
    run_result = None
    while run_result is None:
        try:
            run_result: ApiRun = client.run_pipeline(
                experiment_id=experiment.id,
                job_name=run_name,
                pipeline_package_path=pipeline_file,
                params=arguments)
        except ApiException as e:
            print(f"KFP Server Exception: '{e.reason}' {e.status} '{e.body}' {datetime.datetime.now().strftime('%Y/%m/%d %H:%M:%S')}")
            time.sleep(1)
            client = get_client()

    return run_result.id


def wait_for_run_to_complete(run_id: str) -> ApiRun:
    client = get_client()
    status = None
    while status not in ["Succeeded", "Failed", "Error", "Skipped", "Terminated", "Completed", "CouldntGetTask"]:
        try:
            run_detail: ApiRunDetail = client.get_run(run_id)
            run: ApiRun = run_detail.run
            status = run.status
        except ApiException as e:
            print(f"KFP Server Exception: {e.reason}")
            time.sleep(1)
        time.sleep(0.1)
    print(f"{run.name} took {(run.finished_at - run.created_at)}:"
          f" started at {run.created_at.strftime('%H:%M:%S.%f')[:-3]},"
          f" `{run.status}` at {run.finished_at.strftime('%H:%M:%S.%f')[:-3]}")
    return run


def load_pipeline_functions() -> [Callable]:
    pipeline_functions = []

    from testdata.sequential import sequential_pipeline
    pipeline_functions.append(sequential_pipeline)

    from testdata.condition import flipcoin
    pipeline_functions.append(flipcoin)

    from testdata.compose import save_most_frequent_word
    pipeline_functions.append(save_most_frequent_word)

    from testdata.retry import retry_sample_pipeline
    pipeline_functions.append(retry_sample_pipeline)

    from testdata.loop_static import pipeline as loop_static
    pipeline_functions.append(loop_static)

    # from testdata.condition_custom_task import flipcoin_pipeline
    # pipeline_functions.append(flipcoin_pipeline)

    from testdata.conditions_and_loops import conditions_and_loops
    pipeline_functions.append(conditions_and_loops)

    # from testdata.loop_in_recursion import flipcoin as loop_in_loop
    # pipeline_functions.append(loop_in_loop)

    # TODO: add more pipelines

    # NOTE: loading samples from outside package scope is hacky
    # sys.path.insert(1, '/Users/dummy/projects/kfp-tekton/samples/lightweight-component')
    # from calc_pipeline import calc_pipeline
    # pipeline_functions.append(calc_pipeline)

    return pipeline_functions


def run_performance_test():

    output_file = f"performance_test_times_{dt.now().strftime('%Y-%m-%d_%H-%M-%S')}.csv"
    output_sep = ","

    with open(output_file, "w") as f:
        f.write(output_sep.join(["Pipeline", "Compile", "Submit", "Run"]) + "\n")

    pipeline_functions = load_pipeline_functions()

    for p in pipeline_functions:
        t = [dt.now()]

        pipeline_file = compile_pipeline(p)
        t += [dt.now()]

        run_id = run_pipeline(pipeline_file)
        t += [dt.now()]

        run_details = wait_for_run_to_complete(run_id)  # noqa F841
        t += [dt.now()]

        with open(output_file, "a") as f:
            time_deltas = [str(t[i + 1] - t[i]) for i in range(len(t) - 1)]
            f.write(output_sep.join([p.__name__] + time_deltas) + "\n")


if __name__ == '__main__':
    run_performance_test()

    # client = get_client()
    # experiments = client.list_experiments(namespace='mlx')
    # experiment = client.create_experiment(name='PERF_TESTS', namespace='mlx')
    # experiment = client.get_experiment(experiment_name='PERF_TESTS')
    # runs = client.list_runs(experiment_id=experiment.id, page_size=100)
    # print(experiments)
