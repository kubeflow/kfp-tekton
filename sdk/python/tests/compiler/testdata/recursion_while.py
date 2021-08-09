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

from kfp import dsl, components
from kfp_tekton.compiler import TektonCompiler


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty


def flip_coin_op():
    """Flip a coin and output heads or tails randomly."""
    return components.load_component_from_text("""
    name: flip-coin
    description: flip coin
    outputs:
      - {name: output, type: String}
    implementation:
      container:
        image: python:alpine3.6
        command:
        - sh
        - -c
        args:
        - |
          python -c "import random; result = \'heads\' if random.randint(0,1) == 0 \
          else 'tails'; print(result)" | tee $0
        - {outputPath: output}
    """)()


def print_op(msg: str):
    """Print a message."""
    return components.load_component_from_text("""
    name: print
    description: print msg
    inputs:
      - {name: msg, type: String}
    implementation:
      container:
        image: alpine:3.6
        command:
        - echo
        - {inputValue: msg}
    """)(msg=msg)


@dsl._component.graph_component
def flip_component(flip_result, maxVal):
  with dsl.Condition(flip_result == 'heads'):
    print_flip = print_op(flip_result)
    flipA = flip_coin_op().after(print_flip)
    flip_component(flipA.output, maxVal)


@dsl.pipeline(
    name='recursion-pipeline',
    description='shows how to use graph_component and recursion.'
)
def flipcoin(maxVal: int = 12):
  flip_out = flip_coin_op()
  flip_loop = flip_component(flip_out.output, maxVal)
  print_op('cool, it is over. %s' % flip_out.output).after(flip_loop)


if __name__ == '__main__':
    TektonCompiler().compile(flipcoin, __file__.replace('.py', '.yaml'))
