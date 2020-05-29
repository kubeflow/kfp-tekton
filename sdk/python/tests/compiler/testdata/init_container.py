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


echo = dsl.UserContainer(
    name='echo',
    image='alpine:latest',
    command=['echo', 'bye'])


@dsl.pipeline(name='InitContainer', description='A pipeline with init container.')
def init_container_pipeline():
    dsl.ContainerOp(
        name='hello',
        image='alpine:latest',
        command=['echo', 'hello'],
        init_containers=[echo])


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(init_container_pipeline, __file__.replace('.py', '.yaml'))
