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

WRITE_TEXT_STR = """
name: download-file
description: download file
outputs:
  - {name: data, type: String, description: /tmp/results.txt}
  - {name: underscore_test, type: String, description: /tmp/results.txt}
  - {name: multiple_underscore_test, type: String, description: /tmp/results.txt}
implementation:
  container:
    image: aipipeline/echo-text:latest
    args:
    - -c
    - |
      /echo.sh && cp /tmp/results.txt $0 && cp /tmp/results.txt $1 && cp /tmp/results.txt $2
    - {outputPath: data}
    - {outputPath: underscore_test}
    - {outputPath: multiple_underscore_test}
"""

write_text_op = components.load_component_from_text(WRITE_TEXT_STR)

ECHO2_STR = """
name: echo
description: print the text
inputs:
  - {name: text1, type: String}
implementation:
  container:
    image: library/bash:4.4.23
    command:
    - sh
    - -c
    args:
    - |
      echo "Text 1: $0"
    - {inputValue: text1}
"""

echo2_op = components.load_component_from_text(ECHO2_STR)


@dsl.pipeline(
  name='hidden-output-file-pipeline',
  description='Run a script that passes file to a non configurable path'
)
def hidden_output_file_pipeline(
):
    """A three-step pipeline with the first two steps running in parallel."""

    write_text = write_text_op()

    echo_task = echo2_op(write_text.outputs['data'])


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(hidden_output_file_pipeline, __file__.replace('.py', '.yaml'))
