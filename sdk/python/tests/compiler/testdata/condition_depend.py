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


def false() -> str:
    return 'false'


false_op = components.create_component_from_func(
    false, base_image='python:alpine3.6')


@dsl.pipeline(name='pipeline')
def pipeline(param: int = 10):
    condition_op = false_op()
    cond = condition_op.output
    with dsl.Condition(cond == 'true'):
        op2_template = components.load_component_from_text(op1_yaml)
        op2 = op2_template(1, param)

    op3_template = components.load_component_from_text(op1_yaml)
    op3 = op3_template(1, param)
    op4_template = components.load_component_from_text(op1_yaml)
    op4 = op4_template(1, param)
    op4.after(op2)
    op3.after(op4)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf
  tekton_pipeline_conf = TektonPipelineConf()
  tekton_pipeline_conf.set_tekton_inline_spec(True)
  Compiler().compile(pipeline, __file__.replace('.py', '.yaml'), tekton_pipeline_conf=tekton_pipeline_conf)
