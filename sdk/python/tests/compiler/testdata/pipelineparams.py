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
from kubernetes.client.models import V1EnvVar


@dsl.pipeline(name='pipeline-params', description='A pipeline with multiple pipeline params.')
def pipelineparams_pipeline(tag: str = 'latest', sleep_ms: int = 10):

    echo = dsl.Sidecar(
        name='echo',
        image='hashicorp/http-echo:%s' % tag,
        args=['-text="hello world"'],
    )
    op1 = components.load_component_from_text("""
    name: download
    description: download
    inputs:
      - {name: sleep_ms, type: Integer}
    outputs:
      - {name: data, type: String}
    implementation:
      container:
        image: busy:placeholder
        command:
        - sh
        - -c
        args:
        - |
          sleep $0; wget localhost:5678 -O $1
        - {inputValue: sleep_ms}
        - {outputPath: data}
    """)(sleep_ms)
    op1.container.image = "busy:%s" % tag

    op2 = components.load_component_from_text("""
    name: echo
    description: echo
    inputs:
      - {name: message, type: String}
    implementation:
      container:
        image: library/bash
        command:
        - sh
        - -c
        args:
        - |
          echo $MSG $1
        - {inputValue: message}
    """)(op1.output)
    op2.container.add_env_variable(V1EnvVar(name='MSG', value='pipelineParams: '))


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipelineparams_pipeline, __file__.replace('.py', '.yaml'))
