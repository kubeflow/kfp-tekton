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

"""Example demonstrating how to specify imagepullsecrets to access protected
container registry.
"""

from kfp import dsl, components
from kubernetes import client as k8s_client

GET_FREQUENT_WORD_STR = """
name: get-frequent
description: A get frequent word class representing a component in ML Pipelines
inputs:
  - {name: message, type: String}
outputs:
  - {name: word, type: String}
implementation:
  container:
    image: python:3.6-jessie
    command:
    - sh
    - -c
    args:
    - |
      python -c "from collections import Counter; \
      text = '$0'; print('Input: ' + text); words = Counter(text.split()); \
      print(max(words, key=words.get))" \
      | tee $1
    - {inputValue: message}
    - {outputPath: word}
"""

get_frequent_word_op = components.load_component_from_text(GET_FREQUENT_WORD_STR)


@dsl.pipeline(
    name='save-most-frequent',
    description='Get Most Frequent Word and Save to GCS'
)
def imagepullsecrets_pipeline(
        message: str = "When flies fly behind flies, then flies are following flies."):
    """A pipeline function describing the orchestration of the workflow."""

    counter = get_frequent_word_op(message=message)
    # Call set_image_pull_secrets after get_pipeline_conf().
    dsl.get_pipeline_conf() \
        .set_image_pull_secrets([k8s_client.V1ObjectReference(name="secretA")])
    # also set node_selector
    dsl.get_pipeline_conf().set_default_pod_node_selector('kubernetes.io/os', 'linux')


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(imagepullsecrets_pipeline, __file__.replace('.py', '.yaml'))
