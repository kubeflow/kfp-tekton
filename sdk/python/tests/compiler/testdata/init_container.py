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


echo = dsl.UserContainer(
    name='echo',
    image='alpine:latest',
    command=['echo', 'bye'])

HELLO_STR = """
name: hello
description: A simple component
implementation:
  container:
    image: alpine:latest
    command:
    - echo
    - hello
"""
hello_op = components.load_component_from_text(HELLO_STR)


@dsl.pipeline(name='initcontainer', description='A pipeline with init container.')
def init_container_pipeline():
    hello_task = hello_op()
    hello_task.add_init_container(echo)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(init_container_pipeline, __file__.replace('.py', '.yaml'))
