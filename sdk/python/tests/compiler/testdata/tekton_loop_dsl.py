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

import kfp.dsl as dsl
from kfp import components
from kfp_tekton import tekton

op1_yaml = '''\
name: 'my-in-coop1'
inputs:
- {name: item, type: Integer}
- {name: my_pipe_param, type: Integer}
implementation:
    container:
        image: library/bash:4.4.23
        command: ['sh', '-c']
        args:
        - |
          set -e
          echo op1 "$0" "$1"
        - {inputValue: item}
        - {inputValue: my_pipe_param}
'''


@dsl.pipeline(name='my-pipeline')
def pipeline(my_pipe_param: int = 10):
    loop_args = [1, 2]
    # The DSL above should produce the same result and the DSL in the bottom
    # with dsl.ParallelFor(loop_args, parallelism=1) as item:
    #     op1_template = components.load_component_from_text(op1_yaml)
    #     op1 = op1_template(item, my_pipe_param)
    #     condi_1 = tekton.CEL_ConditionOp(f"{item} == 0").output
    #     with dsl.Condition(condi_1 == 'true'):
    #         tekton.Break()
    with tekton.Loop.sequential(loop_args) as item:
        op1_template = components.load_component_from_text(op1_yaml)
        op1 = op1_template(item, my_pipe_param)
        condi_1 = tekton.CEL_ConditionOp(f"{item} == 1").output
        with dsl.Condition(condi_1 == 'true'):
            tekton.Break()


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
