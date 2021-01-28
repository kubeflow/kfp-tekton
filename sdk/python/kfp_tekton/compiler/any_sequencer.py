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

import uuid
from typing import List
from kfp import dsl

from kfp_tekton.compiler._k8s_helper import sanitize_k8s_name


def after_any(container_ops: List[dsl.ContainerOp]):
    '''
    The function add a flag for any condition handler.
    '''
    tasks_list = []
    for cop in container_ops:
        cop_name = sanitize_k8s_name(cop.name)
        tasks_list.append(cop_name)
    task_list_str = ",".join(tasks_list)

    def _after_components(cop):
        cop.any_sequencer = {"tasks_list": task_list_str}
        return cop
    return _after_components


def generate_any_sequencer(task_list):
    '''
    Generate any sequencer task
    '''
    any_sequencer = {
        "name": "any-sequencer-" + str(uuid.uuid4().hex[:5]),
        "params": [],
        "taskSpec": {
            "steps": [
                {
                    "name": "main",
                    "args": [
                        "-namespace",
                        "$(context.pipelineRun.namespace)",
                        "-prName",
                        "$(context.pipelineRun.name)",
                        "-taskList",
                        task_list
                    ],
                    "command": [
                        "any-taskrun"
                    ],
                    "image": "dspipelines/any-sequencer:latest"
                }
            ]
        }
    }
    return any_sequencer
