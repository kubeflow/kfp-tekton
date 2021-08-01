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

FREQUENT_WORD_STR = """
name: get-frequent
description: Calculate the frequent word from a text
inputs:
  - {name: message, type: String, description: 'Required. message'}
outputs:
  - {name: word, type: String}
implementation:
  container:
    image: python:3.6-jessie
    command:
    - sh
    - -c
    - |
      python -c "import sys; from collections import Counter; \
      input_text = sys.argv[1]; \
      words = Counter(input_text.split()); print(max(words, key=words.get));" \
      "$0" | tee $1
    - {inputValue: message}
    - {outputPath: word}
"""

frequent_word_op = components.load_component_from_text(FREQUENT_WORD_STR)

SAVE_MESSAGE_STR = """
name: save
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
"""

save_message_op = components.load_component_from_text(SAVE_MESSAGE_STR)


@dsl.pipeline(
    name='save-most-frequent',
    description='Get Most Frequent Word and Save to GCS'
)
def save_most_frequent_word(message: str,
                            outputpath: str):
    """A pipeline function describing the orchestration of the workflow."""

    counter = frequent_word_op(message=message)

    saver = save_message_op(
        message=counter.outputs['word'],
        output_path=outputpath)


DOWNLOAD_MESSAGE_STR = """
name: download
description: |
  downloads a message and outputs it
inputs:
  - {name: url, type: String, description: 'Required. the gcs url to download the message from'}
outputs:
  - {name: downloaded, type: String, description: 'file content.'}
implementation:
  container:
    image: google/cloud-sdk
    command:
    - sh
    - -c
    - |
      set -e
      gsutil cat $0 | tee $1
    - {inputValue: url}
    - {outputPath: downloaded}
"""

download_message_op = components.load_component_from_text(DOWNLOAD_MESSAGE_STR)


@dsl.pipeline(
    name='download-and-save-most-frequent',
    description='Download and Get Most Frequent Word and Save to GCS'
)
def download_save_most_frequent_word(
        url: str = 'gs://ml-pipeline-playground/shakespeare1.txt',
        outputpath: str = '/tmp/output.txt'):
    downloader = download_message_op(url)
    save_most_frequent_word(downloader.outputs['downloaded'], outputpath)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    tkc = TektonCompiler()
    tkc.compile(save_most_frequent_word, __file__.replace('.py', '.yaml'))  # Check if simple pipeline can be compiled
    tkc.compile(download_save_most_frequent_word, __file__.replace('.py', '.yaml'))
