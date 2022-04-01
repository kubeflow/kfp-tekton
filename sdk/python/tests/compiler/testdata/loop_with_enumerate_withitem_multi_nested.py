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
from kfp_tekton.tekton import Loop
from typing import List
from kfp import components


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty

op1_yaml = '''\
name: 'my-in-coop1'
inputs:
- {name: index, type: Integer}
- {name: item, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op1 "$0) $1"
        - {inputValue: index}
        - {inputValue: item}
'''

op11_yaml = '''\
name: 'my-1st-inner-coop'
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
          echo "op1-1" "$0" "$1"
        - {inputValue: item}
        - {inputValue: inner_item}
'''

op12_yaml = '''\
name: 'my-2nd-inner-coop'
inputs:
- {name: index, type: Integer}
- {name: outter_index, type: Integer}
- {name: item, type: Integer}
- {name: inner_item, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo "op1-2" "$0)" $1 "$2) $3"
        - {inputValue: outter_index}
        - {inputValue: item}
        - {inputValue: index}
        - {inputValue: inner_item}
'''

op13_yaml = '''\
name: 'my-3rd-inner-coop'
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
          echo "op1-3" "$0" "$1"
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
    with Loop(loop_args).enumerate() as (i, item):
        # 2 iterations in total
        op1_template = components.load_component_from_text(op1_yaml)
        op1 = op1_template(index=i, item=item.a)

        with dsl.ParallelFor(my_pipe_param) as inner_item:
            # 4 iterations in total
            op11_template = components.load_component_from_text(op11_yaml)
            op11 = op11_template(item.a, inner_item)

            my_pipe_param2: List[int] = [4, 5]
            with Loop(my_pipe_param2).enumerate() as (j, inner_item):
                # 8 iterations in total
                op12_template = components.load_component_from_text(op12_yaml)
                op12 = op12_template(outter_index=i, index=j, item=item.b, inner_item=inner_item)
                with dsl.ParallelFor(my_pipe_param3) as inner_item:
                    # 16 iterations in total
                    op13_template = components.load_component_from_text(op13_yaml)
                    op13 = op13_template(item.b, inner_item)

        op2_template = components.load_component_from_text(op2_yaml)
        op2 = op2_template(item.b)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
