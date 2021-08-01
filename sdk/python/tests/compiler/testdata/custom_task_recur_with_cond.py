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
from kfp.components import load_component_from_text
from kfp_tekton.tekton import CEL_ConditionOp
from kfp_tekton.compiler import TektonCompiler


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty

PrintOp = load_component_from_text("""
  name: print
  inputs:
  - name: msg
    type: String
  outputs:
  - name: stdout
    type: String
  implementation:
    container:
      image: alpine:3.6
      command:
      - "sh"
      - "-c"
      args:
      - concat:
        - "echo "
        - { inputValue: msg }
        - " > /tmp/stdout"
      fileOutputs:
        stdout: "/tmp/stdout"
""")


class CEL_Condition(dsl.Condition):
  def __init__(self, pred: str, name: str = None):
    super().__init__(CEL_ConditionOp(pred).output == 'true', name)


def CEL_ExprOp(expr: str):
  return CEL_ConditionOp(expr)


@dsl.graph_component
def recur(i: int, until: int):
  PrintOp(f"Iter: {i}")
  incr_i = CEL_ExprOp(f"{i} + 1").output
  with CEL_Condition(f"{incr_i} < {until}"):
    recur(incr_i, until)


@dsl.pipeline("recursion-test")
def recursion_test(until: int = 5):
  start_idx = CEL_ExprOp("0").output  # graph components require all inputs to be PipelineParams
  with CEL_Condition(f"{start_idx} < {until}"):
    recur(start_idx, until)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  Compiler().compile(recursion_test, __file__.replace('.py', '.yaml'))
