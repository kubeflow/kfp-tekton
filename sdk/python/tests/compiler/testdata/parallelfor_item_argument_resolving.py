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

import kfp
from kfp.components import func_to_container_op
import kfp.dsl as dsl
from kfp.dsl import _for_loop


class Coder:
    def __init__(self, ):
        self._code_id = 0

    def get_code(self, ):
        self._code_id += 1
        return '{code:0{num_chars:}d}'.format(code=self._code_id, num_chars=_for_loop.LoopArguments.NUM_CODE_CHARS)


dsl.ParallelFor._get_unique_id_code = Coder().get_code


@func_to_container_op
def produce_str() -> str:
    return "Hello"


@func_to_container_op
def produce_list_of_dicts() -> list:
    return ({"aaa": "aaa1", "bbb": "bbb1"}, {"aaa": "aaa2", "bbb": "bbb2"})


@func_to_container_op
def produce_list_of_strings() -> list:
    return ("a", "z")


@func_to_container_op
def produce_list_of_ints() -> list:
    return (1234567890, 987654321)


@func_to_container_op
def consume(param1):
    print(param1)


@kfp.dsl.pipeline()
def parallelfor_item_argument_resolving():
    produce_str_task = produce_str()
    produce_list_of_strings_task = produce_list_of_strings()
    produce_list_of_ints_task = produce_list_of_ints()
    produce_list_of_dicts_task = produce_list_of_dicts()

    with kfp.dsl.ParallelFor(produce_list_of_strings_task.output) as loop_item:
        consume(produce_list_of_strings_task.output)
        consume(loop_item)
        consume(produce_str_task.output)

    with kfp.dsl.ParallelFor(produce_list_of_ints_task.output) as loop_item:
        consume(produce_list_of_ints_task.output)
        consume(loop_item)

    with kfp.dsl.ParallelFor(produce_list_of_dicts_task.output) as loop_item:
        consume(produce_list_of_dicts_task.output)
        # consume(loop_item) # Cannot use the full loop item when it's a dict
        consume(loop_item.aaa)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(parallelfor_item_argument_resolving, __file__.replace('.py', '.yaml'))
