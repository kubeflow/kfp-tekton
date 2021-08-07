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


@dsl.pipeline(name='static-loop-pipeline')
def pipeline(my_pipe_param: str = '10'):
    loop_args = [1, 2, 3]
    with dsl.ParallelFor(loop_args) as item:
        op1 = components.load_component_from_text("""
        name: static-loop-inner-op1
        description: static-loop-inner-op1
        inputs:
          - {name: input1, type: Integer}
          - {name: input2, type: String}
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
        """)(input1=item, input2=my_pipe_param)

        op2 = components.load_component_from_text("""
        name: static-loop-inner-op2
        description: static-loop-inner-op2
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
        """)(input1=item)

    op_out = components.load_component_from_text("""
    name: static-loop-out-op
    description: static-loop-out-op
    inputs:
      - {name: input1, type: String}
    implementation:
      container:
        image: library/bash:4.4.23
        command:
        - sh
        - -c
        args:
        - |
          echo $0
        - {inputValue: input1}
    """)(input1=my_pipe_param)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
