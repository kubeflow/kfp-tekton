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


def gcs_download_op(url: str):
    return components.load_component_from_text("""
    name: gcs-download
    description: GCS - Download
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
        args:
        - |
          gsutil cat $0 | tee $1
        - {inputValue: url}
        - {outputPath: data}
    """)(url=url)


def echo2_op(text1: str, text2: str):
    return components.load_component_from_text("""
    name: echo
    description: echo
    inputs:
      - {name: input1, type: String}
      - {name: input2, type: String}
    implementation:
      container:
        image: library/bash:4.4.23
        command:
        - sh
        - -c
        args:
        - |
          echo "Text 1: $0"; echo "Text 2: $1"
        - {inputValue: input1}
        - {inputValue: input2}
    """)(input1=text1, input2=text2)


@dsl.pipeline(
  name='parallel-pipeline',
  description='Download two messages in parallel and prints the concatenated result.'
)
def download_and_join(
    url1: str = 'gs://ml-pipeline-playground/shakespeare1.txt',
    url2: str = 'gs://ml-pipeline-playground/shakespeare2.txt'
):
    """A three-step pipeline with the first two steps running in parallel."""

    download1_task = gcs_download_op(url1)
    download2_task = gcs_download_op(url2)

    echo_task = echo2_op(download1_task.output, download2_task.output)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(download_and_join, __file__.replace('.py', '.yaml'))
