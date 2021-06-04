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


@dsl.pipeline(
    name='nested recursion pipeline',
    description='shows how to use graph_component and nested recursion.'
)
def flipcoin(maxVal=12):
  @dsl._component.graph_component
  def flip_component(flip_result, maxVal):
    @dsl._component.graph_component
    def flip_component_b(flip_result_b, maxVal_b):
      with dsl.Condition(flip_result_b == 'heads'):
        print_flip_b = print_op(flip_result_b)
        flipB = flip_coin_op().after(print_flip_b)
        flip_component_b(flipB.output, maxVal_b)
    recur_b = flip_component_b(flip_result, maxVal)
    print_flip = print_op(flip_result)
    with dsl.Condition(flip_result == 'tails'):
      flipA = flip_coin_op().after(print_flip, recur_b)  # note this part!
      flip_component(flipA.output, maxVal)

  flip_out = flip_coin_op()
  flip_loop = flip_component(flip_out.output, maxVal)
  print_op('cool, it is over. %s' % flip_out.output).after(flip_loop)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  # For Argo, uncomment the line below
  # from kfp.compiler import Compiler
  Compiler().compile(flipcoin, __file__.replace('.py', '.yaml'))
