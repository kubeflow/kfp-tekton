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
from kfp import components


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty

op0_yaml = '''\
name: 'my-out-cop0'
outputs:
  - {name: out, type: String}
implementation:
    container:
        image: python:alpine3.6
        command: ['sh', '-c']
        args:
        - |
          set -e
          python -c "import json; import sys; json.dump([{\'a\': 1, \'b\': 2}, {\'a\': 10, \'b\': 20}], open('$0', 'w'))"
        - {outputPath: out}
'''

op1_yaml = '''\
name: 'my-in-cop1'
inputs:
- {name: item-a, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo no output global op1, item.a: "$0"
        - {inputValue: item-a}
'''

op_out_yaml = '''\
name: 'my-out-cop2'
inputs:
- {name: output, type: String}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo no output global op2, outp: "$0"
        - {inputValue: output}
'''


@dsl.pipeline(name='withparam-output-dict')
def pipeline():
    op0_template = components.load_component_from_text(op0_yaml)
    op0 = op0_template()

    with dsl.ParallelFor(op0.output) as item:
        op1_template = components.load_component_from_text(op1_yaml)
        op1 = op1_template(item.a)

    op_out_template = components.load_component_from_text(op_out_yaml)
    op_out = op_out_template(op0.output)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
