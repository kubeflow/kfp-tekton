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
import kfp

PRINT_STR = """
name: print
description: print a message
inputs:
  - {name: msg, type: String}
implementation:
  container:
    image: alpine:3.6
    command:
    - echo
    - {inputValue: msg}
"""

print_op = components.load_component_from_text(PRINT_STR)

echo_num = 50

@dsl.pipeline(
    name='many-edges-pipeline',
    description='many edges'
)
def manyecho():
    dep_list = []
    for idx in range(echo_num):
      op = print_op("test")
      if len(dep_list) > 0:
        op.after(*dep_list)
      dep_list.append(op)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(manyecho, __file__.replace('.py', '.yaml'))
    # kfp.compiler.Compiler().compile(manyecho, __file__.replace('.py', '-argo.yaml'))
