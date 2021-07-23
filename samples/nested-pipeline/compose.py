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

def frequent_word(message: str) -> str:
    """get frequent word."""
    from collections import Counter
    words = Counter(message.split())
    result = max(words, key=words.get)
    print(result)
    return result

frequent_word_op = components.create_component_from_func(
    func=frequent_word, base_image='python:3.5-jessie')

save_message_op = components.load_component_from_text("""
    name: 'Save Message'
    description: |
      save message to a given output_path
    inputs:
      - {name: message, type: String, description: 'Required. message'}
      - {name: output_path, type: String, description: 'Required. output path'}
    implementation:
      container:
        image: google/cloud-sdk
        command:
        - sh
        - -c
        - |
          set -e
          echo "$0"| gsutil cp - "$1"
        - {inputValue: message}
        - {inputValue: output_path}
    """)

download_message_op = components.load_component_from_text("""
    name: 'Download Message'
    description: |
      downloads a message and outputs it
    inputs:
      - {name: url, type: String, description: 'Required. the gcs url to download the message from'}
    outputs:
      - {name: content, type: String, description: 'file content.'}
    implementation:
      container:
        image: google/cloud-sdk
        command:
        - sh
        - -c
        - |
          set -e
          gsutil cp "$0" "$1"
        - {inputValue: url}
        - {outputPath: content}
    """)

@dsl.pipeline(
    name='save-most-frequent',
    description='Get Most Frequent Word and Save to GCS'
)
def save_most_frequent_word(
              message: str = "hello world hello",
              outputpath: str = "result.txt"):
    """A pipeline function describing the orchestration of the workflow."""
    counter_task = frequent_word_op(message=message)
    content_ref = counter_task.outputs['Output']
    save_message_op(
        message=content_ref,
        output_path=outputpath
    )


@dsl.pipeline(
    name='download-and-save-most-frequent',
    description='Download and Get Most Frequent Word and Save to GCS'
)
def download_save_most_frequent_word(
        url: str= "gs://ml-pipeline-playground/shakespeare1.txt",
        outputpath: str= "res.txt"):
    downloader_task = download_message_op(url)
    save_most_frequent_word(
        message=downloader_task.output,
        outputpath=outputpath)

if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(download_save_most_frequent_word, __file__.replace('.py', '.yaml'))
