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
        image: google/cloud-sdk:216.0.0
        command:
        - sh
        - -c
        args:
        - |
          gsutil cat $0 | tee $1
        - {inputValue: url}
        - {outputPath: data}
    """)(url=url)


def echo_op(text: str):
    return components.load_component_from_text("""
    name: echo
    description: print msg
    inputs:
      - {name: msg, type: String}
    implementation:
      container:
        image: library/bash:4.4.23
        command:
        - sh
        - -c
        args:
        - echo
        - {inputValue: msg}
    """)(msg=text)


@dsl.pipeline(
    name='sequential-pipeline',
    description='A pipeline with two sequential steps.'
)
def sequential_pipeline(
        url: str = 'gs://ml-pipeline-playground/shakespeare1.txt',
        path: str = '/tmp/results.txt'
):
    """A pipeline with two sequential steps."""

    download_task = gcs_download_op(url)
    echo_task = echo_op(path)

    echo_task.after(download_task)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(sequential_pipeline, __file__.replace('.py', '.yaml'))
