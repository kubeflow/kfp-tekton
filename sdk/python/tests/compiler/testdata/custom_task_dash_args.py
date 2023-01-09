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


from kfp import dsl
from kfp.components import load_component_from_text


@dsl.pipeline()
def pipeline_dash_args():
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
              1234567-var-store, --lit_0, '---', --taskSpec,
              '{}']
            fileOutputs: {}
    """)()

    cel_run_bash_script.add_pod_annotation("valid_container", "false")


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf
  tekton_pipeline_conf = TektonPipelineConf()
  tekton_pipeline_conf.set_tekton_inline_spec(True)
  Compiler().compile(pipeline_dash_args, __file__.replace('.py', '.yaml'), tekton_pipeline_conf=tekton_pipeline_conf, type_check=False)
