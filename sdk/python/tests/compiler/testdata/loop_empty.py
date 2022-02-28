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

from kfp import dsl
from kfp_tekton.compiler import TektonCompiler
from kfp_tekton.tekton import Loop


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty


@dsl.pipeline("empty-loop")
def loop_empty(start: int = 0, end: int = 5):
  with Loop.range(start, end):
    pass


if __name__ == '__main__':
  TektonCompiler().compile(loop_empty, __file__.replace('.py', '.yaml'))

