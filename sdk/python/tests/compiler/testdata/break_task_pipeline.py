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

# This use case was discussed in https://github.com/kubeflow/kfp-tekton/issues/845

from kfp import dsl, components
from kfp.components import load_component_from_text
from kfp_tekton import tekton

PrintOp = load_component_from_text("""
  name: print
  inputs:
  - name: msg
  outputs:
  - name: stdout
  implementation:
    container:
      image: alpine:3.6
      command:
      - concat:
        - "echo "
        - { inputValue: msg }
""")

op1_yaml = '''\
name: 'my-in-coop1'
inputs:
- {name: item, type: Integer}
- {name: param, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op1 "$0" "$1"
        - {inputValue: item}
        - {inputValue: param}
'''


@dsl.pipeline(name='pipeline')
def pipeline(param: int = 10):
    loop_args = [1, 2]
    with dsl.ParallelFor(loop_args, parallelism=1) as item:
        op1_template = components.load_component_from_text(op1_yaml)
        op1 = op1_template(item, param)
        condi_1 = tekton.CEL_ConditionOp(f"{item} == 2").output
        with dsl.Condition(condi_1 == 'true'):
            tekton.Break()


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf
  tekton_pipeline_conf = TektonPipelineConf()
  tekton_pipeline_conf.set_tekton_inline_spec(True)
  Compiler().compile(pipeline, __file__.replace('.py', '.yaml'), tekton_pipeline_conf=tekton_pipeline_conf)
