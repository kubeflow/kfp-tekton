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

from kfp import dsl
from kfp import components
from kfp_tekton.compiler import TektonCompiler
from kfp_tekton.tekton import after_any


# Note:
# any_sequencer needs k8s permission to watch the pipelinerun/taskrun
# status. Therefore, make sure the service account that is used to
# run the pipeline has sufficient permission. For example, in
# multi-user deployment, you need to run the follow command to add
# `cluster-admin` cluster-role to the `default-editor` service account
# under your namespace:
#   # assuming the namespace is: mynamespace
#   kubectl create clusterrolebinding pipeline-runner-extend \
#     --clusterrole cluster-admin --serviceaccount=mynamespace:default-editor

def flip_coin() -> str:
    """Flip a coin and output heads or tails randomly."""
    import random
    result = 'heads' if random.randint(0, 1) == 0 else 'tails'
    print(result)
    return result


flip_coin_op = components.create_component_from_func(
    flip_coin, base_image='python:alpine3.6')


component_text = '''\
name: 'sleepComponent'
description: |
  Component for sleep
inputs:
  - {name: seconds, description: 'Sleep for some seconds', default: 10, type: int}
implementation:
  container:
    image: alpine:latest
    command: ['sleep']
    args: [
      {inputValue: seconds},
    ]
'''
sleepOp_template = components.load_component_from_text(component_text)


@dsl.pipeline(
    name="any-sequencer",
    description="Any Sequencer Component Demo",
)
def any_sequence_pipeline(
):
    task1 = sleepOp_template(15)

    task2 = sleepOp_template(200)

    task3 = sleepOp_template(300)

    flip_out = flip_coin_op()

    flip_out.after(task1)

    task4 = sleepOp_template(30)

    task4.apply(after_any([task2, task3, flip_out.outputs['output'] == "heads"], "any_test", 'status'))


if __name__ == "__main__":
    TektonCompiler().compile(any_sequence_pipeline, __file__.replace('.py', '.yaml'))
