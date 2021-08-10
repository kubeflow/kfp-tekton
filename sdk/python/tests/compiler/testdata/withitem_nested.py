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

import kfp.dsl as dsl
from kfp import components
from kfp_tekton.compiler import TektonCompiler


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty

op1_yaml = '''\
name: 'my-in-coop1'
inputs:
- {name: item, type: Integer}
- {name: my_pipe_param, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op1 "$0" "$1"
        - {inputValue: item}
        - {inputValue: my_pipe_param}
'''

op11_yaml = '''\
name: 'my-inner-inner-coop'
inputs:
- {name: item, type: Integer}
- {name: inner_item, type: Integer}
- {name: my_pipe_param, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op11 "$0" "$1" "$2"
        - {inputValue: item}
        - {inputValue: inner_item}
        - {inputValue: my_pipe_param}
'''

op2_yaml = '''\
name: 'my-in-coop2'
inputs:
- {name: item, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op2 "$0"
        - {inputValue: item}
'''

op_out_yaml = '''\
name: 'my-out-cop'
inputs:
- {name: my_pipe_param, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo "$0"
        - {inputValue: my_pipe_param}
'''


@dsl.pipeline(name='my-pipeline')
def pipeline(my_pipe_param: int = 10):
    loop_args = [1, 2]
    with dsl.ParallelFor(loop_args) as item:
        op1_template = components.load_component_from_text(op1_yaml)
        op1 = op1_template(item, my_pipe_param)

        with dsl.ParallelFor([100, 200, 300]) as inner_item:
            op11_template = components.load_component_from_text(op11_yaml)
            op11 = op11_template(item, inner_item, my_pipe_param)

        op2_template = components.load_component_from_text(op2_yaml)
        op2 = op2_template(item)

    op_out_template = components.load_component_from_text(op_out_yaml)
    op_out = op_out_template(my_pipe_param)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
