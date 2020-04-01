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

from kubernetes.client import V1Affinity, V1NodeSelector, V1NodeSelectorRequirement, V1NodeSelectorTerm, V1NodeAffinity
from kfp.dsl import ContainerOp
from kfp import dsl


def some_op():
    return dsl.ContainerOp(
        name='sleep',
        image='busybox',
        command=['sleep 1'],
    )

@dsl.pipeline(
    name='affinity',
    description='A pipeline with affinity'
)
def affinity_pipeline(
):
    """A pipeline with affinity"""
    affinity = V1Affinity(
        node_affinity=V1NodeAffinity(
            required_during_scheduling_ignored_during_execution=V1NodeSelector(
                node_selector_terms=[V1NodeSelectorTerm(
                    match_expressions=[V1NodeSelectorRequirement(
                        key='beta.kubernetes.io/instance-type', operator='In', values=['p2.xlarge'])])])))
    some_op().add_affinity(affinity)

if __name__ == '__main__':
    # don't use top-level import of TektonCompiler to prevent monkey-patching KFP compiler when using KFP's dsl-compile
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(affinity_pipeline, __file__.replace('.py', '.yaml'), generate_pipelinerun=True)
