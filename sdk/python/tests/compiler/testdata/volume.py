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


import kfp.dsl as dsl
from kubernetes.client import V1Volume, V1SecretVolumeSource, V1VolumeMount, V1EnvVar


@dsl.pipeline(
    name='Volume',
    description='A pipeline with volume.'
)
def volume_pipeline():
    op1 = dsl.ContainerOp(name='download',
                          image='google/cloud-sdk',
                          command=['sh', '-c'],
                          arguments=['ls | tee /tmp/results.txt'],
                          file_outputs={'downloaded': '/tmp/results.txt'}) \
        .add_volume(V1Volume(name='gcp-credentials',
                             secret=V1SecretVolumeSource(secret_name='user-gcp-sa'))) \
        .add_volume_mount(V1VolumeMount(mount_path='/secret/gcp-credentials',
                                        name='gcp-credentials')) \
        .add_env_variable(V1EnvVar(name='GOOGLE_APPLICATION_CREDENTIALS',
                                   value='/secret/gcp-credentials/user-gcp-sa.json')) \
        .add_env_variable(V1EnvVar(name='Foo', value='bar'))

    op2 = dsl.ContainerOp(name='echo',
                          image='library/bash',
                          command=['sh', '-c'],
                          arguments=['echo %s' % op1.output])


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(volume_pipeline, __file__.replace('.py', '.yaml'))
