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

from typing import List

from kfp import dsl
from kfp.dsl._container_op import ContainerOp
from kfp_tekton.compiler._k8s_helper import sanitize_k8s_name


class AnySequencer(ContainerOp):
    """A containerOp that will proceed when any of the dependent containerOps completed
       successfully

    Args:
        name: The name of the containerOp. It does not have to be unique within a pipeline
                because the pipeline will generate a unique new name in case of conflicts.

        any: List of `Conditional` containerOps that deploy together with the `main`
                containerOp.
    """
    def __init__(self,
                 any: List[dsl.ContainerOp],
                 name: str = None,):

        tasks_list = []
        for cop in any:
            cop_name = sanitize_k8s_name(cop.name)
            tasks_list.append(cop_name)
        task_list_str = ",".join(tasks_list)

        super().__init__(
            name=name,
            image="dspipelines/any-sequencer:latest",
            command="any-taskrun",
            arguments=[
                        "-namespace",
                        "$(context.pipelineRun.namespace)",
                        "-prName",
                        "$(context.pipelineRun.name)",
                        "-taskList",
                        task_list_str
                    ],
        )


def after_any(any: List[dsl.ContainerOp], name: str = None):
    '''
    The function adds a new AnySequencer and connects the given op to it
    '''
    seq = AnySequencer(any, name)
    
    def _after_components(cop):
        cop.after(seq)
        return cop
    return _after_components
