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

from kfp import dsl
from kfp_tekton.tekton import CEL_ConditionOp
from kfp import components


def flip_coin() -> str:
    """Flip a coin and output heads or tails randomly."""
    import random
    result = 'heads' if random.randint(0, 1) == 0 else 'tails'
    print(result)
    return result

def print_msg(msg: str):
    """Print a message."""
    print(msg)

flip_coin_op = components.create_component_from_func(
    flip_coin, base_image='python:alpine3.6')

print_op = components.create_component_from_func(
    print_msg, base_image='python:alpine3.6')


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
