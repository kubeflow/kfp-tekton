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

from kfp import dsl
from kfp.components import load_component_from_text
from kfp_tekton.tekton import CEL_ConditionOp
from kfp_tekton.compiler import TektonCompiler


class Coder:
  def empty(self):
    return ""


TektonCompiler._get_unique_id_code = Coder.empty


class CelCondition(dsl.Condition):
  def __init__(self, pred: str, name: str = None):
    super().__init__(CEL_ConditionOp(pred).output == 'true', name)


def PrintOp(name: str, msg: str = None):
  if msg is None:
    msg = name
  print_op = load_component_from_text(
  """
  name: %s
  inputs:
  - {name: input_text, type: String, description: 'Represents an input parameter.'}
  outputs:
  - {name: output_value, type: String, description: 'Represents an output paramter.'}
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


@dsl.pipeline("empty-loop")
def condition_1(param: list = ["a", "b", "c"]):
  op0 = PrintOp("print-0")

  with CelCondition(f'{op0.output} == "print-0"'):
    op1 = PrintOp("print-1")

  # works fine for a task outside of loop:
  op2 = PrintOp("print-2")
  op2.after(op1)

  # ...but breaks for loop or task inside of a loop:
  loop = dsl.ParallelFor(param)
  loop2 = dsl.ParallelFor(param)
  # both A) and B) yield the same result:
  # A)
  loop.after(op1)
  with loop:
    op3 = PrintOp("print-3")
  with loop2:
    op4 = PrintOp("print-4")
    # B)
    op4.after(op1)


if __name__ == '__main__':
  TektonCompiler().compile(condition_1, __file__.replace('.py', '.yaml'))
