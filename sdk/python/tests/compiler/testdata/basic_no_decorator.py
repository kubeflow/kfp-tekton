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
# import kfp.gcp as gcp


message_param = dsl.PipelineParam(name='message', value='When flies fly behind flies')
output_path_param = dsl.PipelineParam(name='outputpath', value='default_output')

FREQUENT_WORD_STR = """
name: frequent-word
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
      python -c "from collections import Counter; \
      words = Counter('$0'.split()); print(max(words, key=words.get))" \
      | tee $1
    - {inputValue: message}
    - {outputPath: word}
"""

frequent_word_op = components.load_component_from_text(FREQUENT_WORD_STR)

SAVE_MESSAGE_STR = """
name: save-message
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

EXIT_STR = """
name: exit-handler
description: exit function
implementation:
  container:
    image: python:3.6-jessie
    command:
    - sh
    - -c
    - echo "exit!"
"""

exit_op = components.load_component_from_text(EXIT_STR)


def save_most_frequent_word():
  exit_task = exit_op()
  with dsl.ExitHandler(exit_task):
    counter = frequent_word_op(message=message_param)
    counter.container.set_memory_request('200M')

    saver = save_message_op(
        message=counter.outputs['word'],
        output_path=output_path_param)
    saver.container.set_cpu_limit('0.5')
    # saver.container.set_gpu_limit('2')
    saver.add_node_selector_constraint('kubernetes.io/os', 'linux')
    # saver.apply(gcp.use_tpu(tpu_cores=2, tpu_resource='v2', tf_version='1.12'))


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler
  tkc = TektonCompiler()
  compiled_workflow = tkc._create_workflow(
    save_most_frequent_word,
    'Save Most Frequent Word',
    'Get Most Frequent Word and Save to GCS',
    [message_param, output_path_param],
    None)
  tkc._write_workflow(compiled_workflow, __file__.replace('.py', '.yaml'))
