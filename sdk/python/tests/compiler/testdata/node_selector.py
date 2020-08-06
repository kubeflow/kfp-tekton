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


def echo_op():
    return dsl.ContainerOp(
        name='echo',
        image='busybox',
        command=['sh', '-c'],
        arguments=['echo "Found my node"']
    )


@dsl.pipeline(
    name='node_selector',
    description='A pipeline with Node Selector'
)
def node_selector_pipeline(
):
    """A pipeline with Node Selector"""
    echo_op().add_node_selector_constraint(
        label_name='kubernetes.io/os',
        value='linux')


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(node_selector_pipeline, __file__.replace('.py', '.yaml'),
                             generate_pipelinerun=True)
