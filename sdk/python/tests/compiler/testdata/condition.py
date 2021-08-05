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

FLIP_COIN_STR = """
name: flip
description: Flip a coin and output heads or tails randomly
inputs:
  - {name: forced_result, type: String}
outputs:
  - {name: output, type: String}
implementation:
  container:
    image: python:alpine3.6
    command:
    - sh
    - -c
    - |
      python -c "import random; import sys; forced_result = '$0'; \
      result = 'heads' if random.randint(0,1) == 0 else 'tails'; \
      print(forced_result) if (forced_result == 'heads' or forced_result == 'tails') else print(result)" \
      | tee $1
    - {inputValue: forced_result}
    - {outputPath: output}
"""

flip_coin_op = components.load_component_from_text(FLIP_COIN_STR)

PRINT_STR = """
name: print
description: print a message
inputs:
  - {name: msg, type: String}
implementation:
  container:
    image: alpine:3.6
    command:
    - echo
    - {inputValue: msg}
"""

print_op = components.load_component_from_text(PRINT_STR)


@dsl.pipeline(
    name='flip-coin-example-pipeline',
    description='Shows how to use dsl.Condition.'
)
def flipcoin(forced_result1: str = 'heads', forced_result2: str = 'tails', forced_result3: str = 'heads'):
    flip = flip_coin_op(str(forced_result1))
    flip3 = flip_coin_op(str(forced_result3))

    with dsl.Condition(flip.outputs['output'] == 'heads'):
        flip2 = flip_coin_op(str(forced_result2))

        with dsl.Condition(flip2.outputs['output'] == 'tails'):
            with dsl.Condition(flip3.outputs['output'] == 'heads'):
                print_op(flip2.outputs['output'])

    with dsl.Condition(flip.output == 'tails'):
        print_op(flip.outputs['output'])


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(flipcoin, __file__.replace('.py', '.yaml'))
