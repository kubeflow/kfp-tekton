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
import kfp.components as comp
from typing import NamedTuple
import json


def print4results() -> NamedTuple('taskOutput', [('param1', str), ('param2', str), ('param3', str),
 ('superlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongname', str)]):
   """Print 4 long results"""
   a = 'a' * 2500
   b = 'b' * 700
   c = 'c' * 500
   d = 'd' * 900
   from collections import namedtuple
   task_output = namedtuple('taskOutput', ['param1', 'param2', 'param3',
   'superlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongname'])
   return task_output(a, b, c, d)


print_op = comp.func_to_container_op(print4results)


@dsl.pipeline(
   name='many-results-pipeline',
   description='A pipeline that produce many results.'
)
def many_results_pipeline(
):
    output_estimation_json = {'param1': 2500, 'param23': 700, 'param3': 500,
    'superlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongnamesuperlongname': 900}
    print_task = print_op().add_pod_annotation('tekton-result-sizes', json.dumps(output_estimation_json))


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(many_results_pipeline, __file__.replace('.py', '.yaml'))
