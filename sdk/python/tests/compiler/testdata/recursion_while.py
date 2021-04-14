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
from kfp_tekton.compiler import TektonCompiler


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty


def flip_coin_op():
    """Flip a coin and output heads or tails randomly."""
    return dsl.ContainerOp(
        name='Flip coin',
        image='python:alpine3.6',
        command=['sh', '-c'],
        arguments=['python -c "import random; result = \'heads\' if random.randint(0,1) == 0 '
                  'else \'tails\'; print(result)" | tee /tmp/output'],
        file_outputs={'output': '/tmp/output'}
    )


def print_op(msg):
    """Print a message."""
    return dsl.ContainerOp(
        name='Print',
        image='alpine:3.6',
        command=['echo', msg],
    )


@dsl._component.graph_component
def flip_component(flip_result, maxVal):
  with dsl.Condition(flip_result == 'heads'):
    print_flip = print_op(flip_result)
    flipA = flip_coin_op().after(print_flip)
    flip_component(flipA.output, maxVal)


@dsl.pipeline(
    name='pipeline flip coin',
    description='shows how to use graph_component and recursion.'
)
def flipcoin(maxVal=12):
  flip_out = flip_coin_op()
  flip_loop = flip_component(flip_out.output, maxVal)
  print_op('cool, it is over. %s' % flip_out.output).after(flip_loop)


if __name__ == '__main__':
    TektonCompiler().compile(flipcoin, __file__.replace('.py', '.yaml'))
