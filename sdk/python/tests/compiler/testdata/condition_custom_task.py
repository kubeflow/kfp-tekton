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

RANDOM_NUM_STR = """
name: generate-random-number
description: Generate a random number between low and high.
inputs:
  - {name: low, type: Integer}
  - {name: high, type: Integer}
outputs:
  - {name: output, type: Integer}
implementation:
  container:
    image: python:alpine3.6
    command:
    - sh
    - -c
    - |
      python -c "import random; print(random.randint($0, $1))" | tee $2'
    - {inputValue: low}
    - {inputValue: high}
    - {outputPath: output}
"""

random_num_op = components.load_component_from_text(RANDOM_NUM_STR)

FLIP_COIN_STR = """
name: flip-coin
description: Flip a coin and output heads or tails randomly
outputs:
  - {name: output, type: Integer}
implementation:
  container:
    image: python:alpine3.6
    command:
    - sh
    - -c
    - |
      python -c "import random; result = 'heads' if random.randint(0,1) == 0 else 'tails'; print(result)" \
      | tee $0
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
    name='conditional-execution-pipeline',
    description='Shows how to use dsl.Condition().'
)
def flipcoin_pipeline():
    flip = flip_coin_op()
    cel_condition = CEL_ConditionOp("'%s' == 'heads'" % flip.outputs['output'])
    with dsl.Condition(cel_condition.output == 'true'):
        random_num_head = random_num_op(0, 9)
        cel_condition_2 = CEL_ConditionOp("%s > 5" % random_num_head.outputs['output'])
        with dsl.Condition(cel_condition_2.output == 'true'):
            print_op('heads and %s > 5!' % random_num_head.outputs['output'])
        with dsl.Condition(cel_condition_2.output != 'true'):
            print_op('heads and %s <= 5!' % random_num_head.outputs['output'])

    with dsl.Condition(cel_condition.output != 'true'):
        random_num_tail = random_num_op(10, 19)
        cel_condition_3 = CEL_ConditionOp("%s > 15" % random_num_tail.outputs['output'])
        with dsl.Condition(cel_condition_3.output == 'true'):
            inner_task = print_op('tails and %s > 15!' % random_num_tail.outputs['output'])
        with dsl.Condition(cel_condition_3.output != 'true'):
            print_op('tails and %s <= 15!' % random_num_tail.outputs['output'])


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(flipcoin_pipeline, __file__.replace('.py', '.yaml'))
