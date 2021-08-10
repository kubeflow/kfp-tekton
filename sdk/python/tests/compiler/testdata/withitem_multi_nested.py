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
from kfp_tekton.compiler import TektonCompiler
from typing import List
from kfp import components


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty

op1_yaml = '''\
name: 'my-in-coop1'
inputs:
- {name: item, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op1 "$0"
        - {inputValue: item}
'''

op11_yaml = '''\
name: 'my-inner-inner-coop'
inputs:
- {name: item, type: Integer}
- {name: inner_item, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op11 "$0" "$1"
        - {inputValue: item}
        - {inputValue: inner_item}
'''

op12_yaml = '''\
name: 'my-inner-inner-coop'
inputs:
- {name: item, type: Integer}
- {name: inner_item, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op12 "$0" "$1"
        - {inputValue: item}
        - {inputValue: inner_item}
'''

op13_yaml = '''\
name: 'my-inner-inner-coop'
inputs:
- {name: item, type: Integer}
- {name: inner_item, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op13 "$0" "$1"
        - {inputValue: item}
        - {inputValue: inner_item}
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


@dsl.pipeline(name='withitem-multiple-nesting-pipeline')
def pipeline(my_pipe_param: list = [100, 200], my_pipe_param3: list = [1, 2]):
    loop_args = [{'a': 1, 'b': 2}, {'a': 10, 'b': 20}]
    with dsl.ParallelFor(loop_args) as item:
        op1_template = components.load_component_from_text(op1_yaml)
        op1 = op1_template(item.a)

        with dsl.ParallelFor(my_pipe_param) as inner_item:
            op11_template = components.load_component_from_text(op11_yaml)
            op11 = op11_template(item.a, inner_item)

            my_pipe_param2: List[int] = [4, 5]
            with dsl.ParallelFor(my_pipe_param2) as inner_item:
                op12_template = components.load_component_from_text(op12_yaml)
                op12 = op11_template(item.b, inner_item)
                with dsl.ParallelFor(my_pipe_param3) as inner_item:
                    op13_template = components.load_component_from_text(op13_yaml)
                    op13 = op11_template(item.b, inner_item)

        op2_template = components.load_component_from_text(op2_yaml)
        op2 = op2_template(item.b)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
