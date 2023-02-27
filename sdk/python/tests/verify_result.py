#!/usr/bin/env python3

# Copyright 2022 kubeflow.org
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
from typing import List, Tuple
import json
import click
from datetime import timedelta
from datetime import datetime as dt
from kubernetes import client, config

from kfp_server_api import ApiRunDetail
from kfp_tekton._client import TektonClient

elapsed_threshold = 1.5


def get_client(ip: str = "localhost", port: str = "8888", ns: str = "kubeflow") -> Tuple[client.CoreV1Api, TektonClient]:
    config.load_kube_config()
    host = f"http://{ip}:{port}"
    tclient = TektonClient(host=host)
    tclient.set_user_namespace(ns)

    return client.CoreV1Api(), tclient


def query_pipelinerun(tclient: TektonClient, runid: str, namespace: str) -> dict:
    run_details: ApiRunDetail = tclient.get_run(runid)

    return json.loads(run_details.to_dict()["pipeline_runtime"]["workflow_manifest"])


def parse_pr(pr: dict) -> dict:
    rev = {}
    status = pr['status']
    dt_fmt = "%Y-%m-%dT%H:%M:%SZ"

    def get_details(data):
        info = {}
        total = timedelta(0)
        count = 0

        items = {}
        for key in data.keys():
            run = data[key]
            status = run['status']
            conditions = status['conditions']
            state = conditions[len(conditions) - 1]['type']
            start = dt.strptime(status['startTime'], dt_fmt)
            end = dt.strptime(status['completionTime'], dt_fmt)
            elapsed = end - start
            items[run['pipelineTaskName']] = {
                'start': start,
                'end': end,
                'elapsed': elapsed,
                'status': state,
            }
            if status['podName']:
                items[run['pipelineTaskName']]['pod'] = status['podName']
            count += 1
            total += elapsed

        info['count'] = count
        info['total_elapsed'] = total
        info['items'] = items
        return info

    if 'taskRuns' in status:
        rev['taskRuns'] = get_details(status['taskRuns'])

    if 'runs' in status:
        rev['runs'] = get_details(status['runs'])

    rev['start'] = dt.strptime(status['startTime'], dt_fmt)
    rev['end'] = dt.strptime(status['completionTime'], dt_fmt)
    return rev


def check_cache(results: dict, corev1: client.CoreV1Api, namespace: str):
    taskruns = results['taskRuns']
    if taskruns:
        items = taskruns['items']
        for taskrun in items:
            logs: str = corev1.read_namespaced_pod_log(name=items[taskrun]['pod'], namespace=namespace, container='step-main')
            if logs.find('This step output is taken from cache') >= 0:
                items[taskrun]['cached'] = True


def calculate_sequences(results) -> List:
    taskruns = results['taskRuns']['items'] if 'taskRuns' in results else {}
    runs = results['runs']['items'] if 'runs' in results else {}
    tasks = {**taskruns, **runs}
    seq_list: list = list(tasks.keys())

    def delta_to_complete(n):
        return results['end'] - tasks[n]['start']

    # sort by start time
    seq_list.sort(reverse=True, key=delta_to_complete)

    return seq_list


def compare(results: dict, seq: list, expected: dict):
    if 'sequence' in expected:
        if seq != expected['sequence']:
            raise Exception("sequence of taskruns/runs is incorrect")

    if 'taskRuns' in expected:
        assert 'taskRuns' in results, 'expect TaskRuns but not found'
        assert set(expected['taskRuns'].keys()) == set(results["taskRuns"]["items"].keys()), "TaskRuns list doesn't match"

        taskruns = expected['taskRuns']
        for taskrun in taskruns:
            if 'cached' in taskruns[taskrun] and taskruns[taskrun]['cached']:
                assert 'cached' in results['taskRuns']['items'][taskrun], f"task: {taskrun} was not cached"
            else:
                assert 'cached' not in results['taskRuns']['items'][taskrun], f"task: {taskrun} was cached"

            elapsed = results['taskRuns']['items'][taskrun]['elapsed']
            exp_elapsed = timedelta(seconds=taskruns[taskrun]['elapsed'])
            assert elapsed <= elapsed_threshold * exp_elapsed, \
                f"TaskRun:{taskrun} elapsed time({elapsed}) is longer than expected duration({elapsed_threshold * exp_elapsed})"

    else:
        assert 'taskRuns' not in results, 'expect no TaskRuns'

    if 'runs' in expected:
        assert 'runs' in results, 'expect Runs but not found'
        assert set(expected['runs'].keys()) == set(results['runs']['items'].keys()), "Runs list doesn't match"

        runs = expected['runs']
        for run in runs:
            elapsed = results['runs']['items'][run]['elapsed']
            exp_elapsed = timedelta(seconds=runs[run]['elapsed'])
            assert elapsed <= elapsed_threshold * exp_elapsed, \
                f"Run:{run} elapsed time({elapsed}) is longer than expected duration({elapsed_threshold * exp_elapsed})"

    else:
        assert 'runs' not in results, 'expect no Runs'


@click.command()
@click.argument('runid')
@click.argument('expect')
@click.option('--ip', default='localhost', help='ip address for the kfp api endpoint')
@click.option('--port', default='8888', help='port of the kfp api endpoint')
@click.option('--namespace', default='kubeflow', help='namespace for the pipelinerun')
@click.option('--threshold', default=1.5, help='threshold for elapsed time')
@click.option('--dump', default=False, help='print out the taskruns/runs info')
def verify_pipelinerun(runid, expect, ip, port, namespace, threshold, dump):
    global elapsed_threshold
    elapsed_threshold = threshold
    corev1, tclient = get_client(ip, port, namespace)
    pr = query_pipelinerun(tclient, runid, namespace)
    assert pr is not None, f"can't find the pipelinerun for runid:{runid} in namespace: {namespace}"

    with open(expect) as f:
        expected = json.load(f)

    results = parse_pr(pr)
    check_cache(results, corev1, namespace)
    seq = calculate_sequences(results)
    if dump:
        print(results)
    compare(results, seq, expected)


if __name__ == '__main__':
    logging.basicConfig(format='%(message)s', level=logging.INFO)
    verify_pipelinerun()
