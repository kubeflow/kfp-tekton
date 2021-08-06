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

from kfp import dsl
import kfp.components as comp
from kfp_tekton import TektonClient
from kubernetes import client as k8s_client


@comp.create_component_from_func
def print_workflow_info():
    import os
    print("Running workflow with name ", os.getenv('KFP_RUN_NAME') + " id " + os.getenv('KFP_RUN_ID'))


@dsl.pipeline(
    name="k8s-downstream-api",
    description="A basic example using Kubernete downstream API to get KFP run_name and run_id."
)
def downstream_api():
    print_workflow_info().add_env_variable(k8s_client.V1EnvVar(
        name='KFP_RUN_NAME',
        value_from=k8s_client.V1EnvVarSource(
            field_ref=k8s_client.V1ObjectFieldSelector(
                field_path="metadata.annotations['pipelines.kubeflow.org/run_name']"
            )
        )
    )).add_env_variable(k8s_client.V1EnvVar(
        name='KFP_RUN_ID',
        value_from=k8s_client.V1EnvVarSource(
            field_ref=k8s_client.V1ObjectFieldSelector(
                field_path="metadata.labels['pipeline/runid']"
            )
        )
    ))


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(downstream_api, __file__.replace('.py', '.yaml'))
