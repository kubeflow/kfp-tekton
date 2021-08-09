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

from kfp import dsl, components
from kfp_tekton.tekton import CEL_ConditionOp


def flip_coin_op():
    """Flip a coin and output heads or tails randomly."""
    return components.load_component_from_text("""
    name: flip-coin
    description: Flip coin
    outputs:
      - {name: output_result, type: String}
    implementation:
      container:
        image: python:alpine3.6
        command:
        - sh
        - -c
        args:
        - |
          python -c "import random; result = 'heads' if random.randint(0,1) == 0 \
          else 'tails'; print(result)" | tee $0
        - {outputPath: output_result}
    """)()


def print_op(msg: str):
    """Print a message."""
    return components.load_component_from_text("""
    name: print
    description: Print
    inputs:
      - {name: msg, type: String}
    implementation:
      container:
        image: alpine:3.6
        command:
        - echo
        - {inputValue: msg}
    """)(msg=msg)


@dsl.pipeline(
    name='tekton-custom-task-on-kubeflow-pipeline',
    description='Shows how to use Tekton custom task with KFP'
)
def custom_task_pipeline():
    flip = flip_coin_op()
    flip2 = flip_coin_op()
    cel_condition = CEL_ConditionOp("'%s' == '%s'" % (flip.output, flip2.output))
    print_op('Condition output is %s' % cel_condition.output)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(custom_task_pipeline, __file__.replace('.py', '.yaml'))
