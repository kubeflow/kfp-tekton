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
from kfp_tekton.compiler import TektonCompiler as Compiler
from kfp_tekton.tekton import CEL_ConditionOp


def gcs_download_op(url):
    return components.load_component_from_text("""
      name: %s
      description: download
      inputs:
        - {name: url, type: String}
      outputs:
        - {name: %s, type: String}
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
          - {outputPath: %s}
    """ % ('GCS - Download' * 4, 'Data-_123' * 100, 'Data-_123' * 100))(url=url)


@dsl.pipeline(name="some-very-long-name-with-lots-of-words-in-it-" +
                   "it-should-be-over-63-chars-long-in-order-to-observe-the-problem")
def main_fn(url1: str = 'gs://ml-pipeline-playground/shakespeare1.txt'):
    download1_task = gcs_download_op(url1)
    printop = components.load_component_from_text("""
      name: %s
      description: print
      inputs:
        - {name: text, type: String}
      implementation:
        container:
          image: alpine:3.6
          command:
          - echo
          - {inputValue: text}
    """ % ('print' * 10))(download1_task.outputs['Data-_123' * 100])
    cel_condition = CEL_ConditionOp("'%s' == 'heads'" % download1_task.outputs['Data-_123' * 100])


if __name__ == '__main__':
    Compiler().compile(main_fn, __file__.replace('.py', '.yaml'))
