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

from kfp import dsl, components
from kfp_tekton.tekton import CEL_ConditionOp
from kfp_tekton.compiler import TektonCompiler


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty

PRINT_STR = """
name: print-iter
description: print message
inputs:
  - {name: msg, type: String, description: 'Required. message'}
outputs:
  - {name: stdout, type: String}
implementation:
  container:
    image: alpine:3.6
    command:
    - echo
    - {inputValue: msg}
    - ">"
    - {outputPath: stdout}
"""

print_op = components.load_component_from_text(PRINT_STR)


class CelCondition(dsl.Condition):
  def __init__(self, pred: str, name: str = None):
    super().__init__(CEL_ConditionOp(pred).output == 'true', name)


@dsl.graph_component
def recur(i: int):
  decr_i = CEL_ConditionOp(f"{i} - 1").output
  print_op(msg=f"Iter: {decr_i}")
  recur(decr_i)


@dsl.pipeline("condition-and-recur")
def condition_and_recur(iter_num: int = 42):
  with CelCondition(f"{iter_num} != 0"):
    recur(iter_num)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  Compiler().compile(condition_and_recur, __file__.replace('.py', '.yaml'))
