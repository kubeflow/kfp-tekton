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


@dsl.pipeline(
  name='artifact-out-pipeline',
  description='Add labels to identify outputs as artifacts.'
)
def artifact_outputs(
    url1: str = 'gs://ml-pipeline-playground/shakespeare1.txt'
):
    """Add labels to identify outputs as artifacts."""

    download1_task = gcs_download_op(url1).add_pod_label(name='output_type', value='artifact')


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(artifact_outputs, __file__.replace('.py', '.yaml'))
