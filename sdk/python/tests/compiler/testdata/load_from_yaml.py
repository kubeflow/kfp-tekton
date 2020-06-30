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

from kfp import dsl
from kfp import components

component_text = '''\
name: 'busybox'
description: |
  Sample component for testing
inputs:
  - {name: dummy_text,        description: 'Test empty input', default: ''}
outputs:
  - {name: dummy_output_path, description: 'Test unused output path'}
implementation:
  container:
    image: alpine:latest
    command: ['echo']
    args: [
      start,
      {inputValue: dummy_text},
      {outputPath: dummy_output_path},
      end
    ]
'''

test_op = components.load_component_from_text(component_text)


@dsl.pipeline(
    name='component-yaml-pipeline',
    description='A pipeline with components loaded from yaml'
)
def component_yaml_pipeline(
):
    """A pipeline with components loaded from yaml"""
    test = test_op()


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(component_yaml_pipeline, __file__.replace('.py', '.yaml'))
