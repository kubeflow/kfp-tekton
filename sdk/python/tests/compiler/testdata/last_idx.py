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


from kfp_tekton.compiler import TektonCompiler
from kfp import dsl
from kfp.components import load_component_from_text
from kfp_tekton import tekton

PrintOp = load_component_from_text("""
  name: print
  inputs:
  - name: msg
  implementation:
    container:
      image: alpine:3.6
      command:
      - sh
      - -c
      - |
        set -e
        echo $0
      - {inputValue: msg }
""")


@dsl.pipeline(name='pipeline')
def pipeline(param: int = 10):
    loop_args = "1,2"
    loop = tekton.Loop.from_string(loop_args, separator=',')
    with loop.enumerate() as (idx, el):
        # loop enter-local variables, not available outside of the loop
        PrintOp(f"it no {idx}: {el}")
    # using loop's fields, not enter-local variables
    PrintOp(f"last it: no {loop.last_idx}: {loop.last_elem}")


if __name__ == "__main__":
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
