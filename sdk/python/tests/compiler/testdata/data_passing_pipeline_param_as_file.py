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

import kfp
from kfp.components import create_component_from_func, InputPath
from kfp_tekton.compiler import TektonCompiler


# Consume as file
@create_component_from_func
def consume_anything_as_file(data_path: InputPath()):
    with open(data_path) as f:
        print("consume_anything_as_file: " + f.read())


@create_component_from_func
def consume_something_as_file(data_path: InputPath('Something')):
    with open(data_path) as f:
        print("consume_something_as_file: " + f.read())


@create_component_from_func
def consume_string_as_file(data_path: InputPath(str)):
    with open(data_path) as f:
        print("consume_string_as_file: " + f.read())


# Pipeline
@kfp.dsl.pipeline(name='data_passing_pipeline')
def data_passing_pipeline(
    anything_param="anything_param",
    something_param: "Something" = "something_param",
    string_param: str = "string_param",
):

    # Pass pipeline parameter; consume as file
    consume_anything_as_file(anything_param)
    consume_anything_as_file(something_param)
    consume_anything_as_file(string_param)
    consume_something_as_file(anything_param)
    consume_something_as_file(something_param)
    consume_string_as_file(anything_param)
    consume_string_as_file(string_param)


if __name__ == "__main__":
    kfp_endpoint = None
    TektonCompiler().compile(data_passing_pipeline, __file__.replace('.py', '.yaml'))
