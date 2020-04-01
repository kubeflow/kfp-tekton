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

from kubernetes.client import V1Toleration
from kfp.dsl import ContainerOp
from kfp import dsl

@dsl.pipeline(
    name='tolerations',
    description='A pipeline with tolerations'
)
def tolerations(
):
    """A pipeline with tolerations"""
    op1 = dsl.ContainerOp(
        name='download',
        image='busybox',
        command=['sh', '-c'],
        arguments=['sleep 10; wget localhost:5678 -O /tmp/results.txt'],
        file_outputs={'downloaded': '/tmp/results.txt'})\
        .add_toleration(V1Toleration(effect='NoSchedule',
                                     key='gpu',
                                     operator='Equal',
                                     value='run'))

if __name__ == '__main__':
    # don't use top-level import of TektonCompiler to prevent monkey-patching KFP compiler when using KFP's dsl-compile
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(tolerations, __file__.replace('.py', '.yaml'), generate_pipelinerun=True)
