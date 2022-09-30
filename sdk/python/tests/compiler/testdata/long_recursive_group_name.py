# Copyright 2021 kubeflow.org
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
from kfp_tekton.tekton import CEL_ConditionOp
from kfp import components
from kfp_tekton.compiler import TektonCompiler


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty


class CelCondition(dsl.Condition):
  def __init__(self, pred: str, name: str = None):
    super().__init__(CEL_ConditionOp(pred).output == 'true', name)


def PrintOp(name: str, msg: str):
    print_op = components.load_component_from_text(
    """
    name: %s
    inputs:
    - {name: input_text, type: String, description: 'Represents an input parameter.'}
    outputs:
    - {name: output_value, type: String, description: 'Represents an output parameter.'}
    implementation:
        container:
            image: alpine:3.6
            command:
            - sh
            - -c
            - |
              set -e
              echo $0 > $1
            - {inputValue: input_text}
            - {outputPath: output_value}
    """ % (name)
    )
    return print_op(msg)


@dsl.graph_component
def function_the_name_of_which_is_exactly_51_chars_long(i: int):
  decr_i = CEL_ConditionOp(f"{i} - 1").output
  PrintOp("print-iter", f"Iter: {decr_i}")
  with dsl.Condition(decr_i != 0):
    function_the_name_of_which_is_exactly_51_chars_long(decr_i)


@dsl.pipeline("pipeline-the-name-of-which-is-exactly-51-chars-long")
def pipeline_the_name_of_which_is_exactly_51_chars_long(iter_num: int = 42):
  function_the_name_of_which_is_exactly_51_chars_long(iter_num)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf
  tekton_pipeline_conf = TektonPipelineConf()
  tekton_pipeline_conf.set_tekton_inline_spec(False)
  tekton_pipeline_conf.set_resource_in_separate_yaml(True)
  Compiler().compile(pipeline_the_name_of_which_is_exactly_51_chars_long,
    __file__.replace('.py', '.yaml'), tekton_pipeline_conf=tekton_pipeline_conf)
