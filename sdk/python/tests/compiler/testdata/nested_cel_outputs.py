# Copyright 2023 kubeflow.org
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
from kfp.components import load_component_from_text
from kfp_tekton.tekton import Loop

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
def pipeline():

    cel_run_bash_script = load_component_from_text(r"""
        name: cel-run-bash-script
        inputs: []
        outputs:
        - {name: env-variables}
        implementation:
          container:
            image: aipipeline/cel-eval:latest
            command: [cel]
            args: [--apiVersion, custom.tekton.dev/v1alpha1, --kind, VariableStore, --name,
              1234567-var-store, --lit_0, 'a', --taskSpec,
              '{}']
            fileOutputs: {}
    """)()
    with Loop(cel_run_bash_script.output) as item:
        with Loop(cel_run_bash_script.output) as item2:
            op1_template = components.load_component_from_text(op1_yaml)
            op1 = op1_template(1, 10)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf
  tekton_pipeline_conf = TektonPipelineConf()
  tekton_pipeline_conf.set_tekton_inline_spec(True)
  tekton_pipeline_conf.set_resource_in_separate_yaml(True)
  Compiler().compile(pipeline, __file__.replace('.py', '.yaml'), tekton_pipeline_conf=tekton_pipeline_conf)

