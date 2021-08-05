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

GCS_DOWNLOAD_STR = """
name: gcs-download
description: download file from GCS
inputs:
  - {name: url, type: String}
outputs:
  - {name: data, type: String}
implementation:
  container:
    image: google/cloud-sdk:279.0.0
    command:
    - sh
    - -c
    - gsutil cat $0 | tee $1
    - {inputValue: url}
    - {outputPath: data}
"""

gcs_download_op = components.load_component_from_text(GCS_DOWNLOAD_STR)

ECHO_STR = """
name: echo
description: print out message
inputs:
  - {name: text, type: String}
implementation:
  container:
    image: library/bash:4.4.23
    command:
    - sh
    - -c
    - echo "$0"
    - {inputValue: text}
"""

echo_op = components.load_component_from_text(ECHO_STR)


@dsl.pipeline(
    name='exit-handler',
    description='Downloads a message and prints it. The exit handler will run after the pipeline finishes (successfully or not).'
)
def download_and_print(url: str = 'gs://ml-pipeline-playground/shakespeare1.txt'):
    """A sample pipeline showing exit handler."""

    exit_task = echo_op('exit!')

    with dsl.ExitHandler(exit_task):
        download_task = gcs_download_op(url)
        echo_task = echo_op(download_task.outputs['data'])


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(download_and_print,
                             __file__.replace('.py', '.yaml'))
