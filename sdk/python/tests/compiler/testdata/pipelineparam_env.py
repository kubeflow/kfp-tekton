# Copyright 2022 kubeflow.org
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


from kubernetes.client.models import V1EnvVar
from kfp import dsl


def echo_op(port_number):
    return dsl.ContainerOp(
        name='echo',
        image='busybox',
        command=['sh', '-c'],
        arguments=['echo "Got scheduled"']
    ).container.add_env_variable(V1EnvVar(name='PORT', value=port_number))


@dsl.pipeline(
    name='echo',
    description='echo pipeline'
)
def echo_pipeline(
    port_number=80
):
    echo = echo_op(port_number)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(echo_pipeline, 'echo_pipeline.yaml')
