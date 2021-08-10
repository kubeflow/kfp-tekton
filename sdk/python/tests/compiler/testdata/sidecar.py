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


@dsl.pipeline(name='sidecar', description='A pipeline with sidecars.')
def sidecar_pipeline():

    echo = dsl.Sidecar(
        name='echo',
        image='hashicorp/http-echo',
        args=['-text="hello world"'])

    op1 = components.load_component_from_text("""
    name: download
    description: download
    outputs:
      - {name: download, type: String}
    implementation:
      container:
        image: busybox
        command:
        - sh
        - -c
        args:
        - |
          sleep 10; wget localhost:5678 -O $0
        - {outputPath: download}
    """)()

    op2 = components.load_component_from_text("""
    name: echo
    description: echo
    inputs:
      - {name: msg, type: String}
    implementation:
      container:
        image: library/bash
        command:
        - sh
        - -c
        args:
        - echo
        - {inputValue: msg}
    """)(msg=op1.output)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(sidecar_pipeline, __file__.replace('.py', '.yaml'))
