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

from typing import List
from kfp import dsl
from kfp_tekton.compiler import TektonCompiler as Compiler


class Coder:
    def empty(self):
        return ""


Compiler._get_unique_id_code = Coder.empty


class PrintOp(dsl.ContainerOp):
    def __init__(self, name, msg):
        super(PrintOp, self).__init__(
            name=name,
            image='alpine:3.6',
            command=['echo', msg])


@dsl.pipeline(name="Some very long name with lots of words in it. " +
                   "It should be over 63 chars long in order to observe the problem.")
def main_fn(arr: List[str] = ["a", "b", "c"]):
    with dsl.ParallelFor(arr) as it:
        PrintOp('print', it)


if __name__ == '__main__':
    Compiler().compile(main_fn, __file__.replace('.py', '.yaml'))
