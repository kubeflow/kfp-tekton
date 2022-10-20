# Copyright 2022 kubeflow.org
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
from kfp_tekton.tekton import Loop


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


@dsl.pipeline(name='my-pipeline')
def pipeline(my_pipe_param: int = 10, start: int = 1, end: int = 2):
    start_2 = 1
    end_2 = 2
    iterate_param_pass_style_field = {'iterate_param_pass_style': 'inline'}
    item_pass_style_field = {'item_pass_style': 'file'}
    with Loop.range(start=start, end=end, extra_fields=iterate_param_pass_style_field) as item:
        op1_template = components.load_component_from_text(op1_yaml)
        op1_template(item, my_pipe_param)
        with Loop.range(start=start_2, end=end_2, extra_fields=item_pass_style_field) as item2:
            op1_template(item2, my_pipe_param)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler

    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
