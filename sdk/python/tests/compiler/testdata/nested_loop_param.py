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
from kfp_tekton.tekton import Loop
from kfp_tekton.compiler import TektonCompiler


class Coder:
  def empty(self):
    return ""


TektonCompiler._get_unique_id_code = Coder.empty


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


@dsl.pipeline("loop-3-range")
def loop_3_range():
  with Loop.from_string('a,b,c', separator=',') as it0:
    with Loop.from_string('[A,B,C]') as it1:
      with Loop.from_string('1,2,3', separator=',') as it2:
        PrintOp("print", f"print {it0} {it1} {it2}")


if __name__ == '__main__':
  TektonCompiler().compile(loop_3_range, __file__.replace('.py', '.yaml'))
