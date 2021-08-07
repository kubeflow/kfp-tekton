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
            python -c "import random; result = 'heads' if random.randint(0,1) == 0 \
            else 'tails'; print(result)" | tee $0
          - {outputPath: output}
    """)()


def print_op(msg):
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
def flip_component(flip_result: str, maxVal: int, my_pipe_param: int):
  with dsl.Condition(flip_result == 'heads'):
    print_flip = print_op(flip_result)
    flipA = flip_coin_op().after(print_flip)
    loop_args = [{'a': 1, 'b': 2}, {'a': 10, 'b': 20}]
    with dsl.ParallelFor(loop_args) as item:
        op1 = components.load_component_from_text("""
        name: my-in-coop1
        description: in coop1
        inputs:
          - {name: input1, type: Integer}
          - {name: input2, type: Integer}
        implementation:
          container:
            image: library/bash:4.4.23
            command:
            - sh
            - -c
            args:
            - |
              echo op1 $0 $1
            - {inputValue: input1}
            - {inputValue: input2}
        """)(input1=item.a, input2=my_pipe_param)

        with dsl.ParallelFor([100, 200, 300]) as inner_item:
            op11 = components.load_component_from_text("""
            name: my-inner-inner-coop
            description: my inner inner coop
            inputs:
              - {name: input1, type: Integer}
              - {name: input2, type: Integer}
              - {name: input3, type: Integer}
            implementation:
              container:
                image: library/bash:4.4.23
                command:
                - sh
                - -c
                args:
                - |
                  echo op1 $0 $1 $2
                - {inputValue: input1}
                - {inputValue: input2}
                - {inputValue: input3}
            """)(input1=item.a, input2=inner_item, input3=my_pipe_param)

        op2 = components.load_component_from_text("""
        name: my-in-coop2
        description: my in coop2
        inputs:
          - {name: input1, type: Integer}
        implementation:
          container:
            image: library/bash:4.4.23
            command:
            - sh
            - -c
            args:
            - |
              echo op2 $0
            - {inputValue: input1}
        """)(input1=item.b)

    flip_component(flipA.output, maxVal, my_pipe_param)


@dsl.pipeline(
    name='loop-in-recursion-pipeline',
    description='shows how to use graph_component and recursion.'
)
def flipcoin(maxVal: int = 12, my_pipe_param: int = 10):
  flip_out = flip_coin_op()
  flip_loop = flip_component(flip_out.outputs['output'], maxVal, my_pipe_param)
  print_op('cool, it is over. %s' % flip_out.outputs['output']).after(flip_loop)


if __name__ == '__main__':
    TektonCompiler().compile(flipcoin, __file__.replace('.py', '.yaml'))
