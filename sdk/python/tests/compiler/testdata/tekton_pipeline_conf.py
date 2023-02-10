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

from kfp import dsl, components
import kfp_tekton
from kubernetes.client import V1SecurityContext
from kubernetes.client.models import V1Volume, V1PersistentVolumeClaimVolumeSource, \
    V1PersistentVolumeClaimSpec, V1ResourceRequirements


def echo_op():
    return components.load_component_from_text("""
    name: echo
    description: echo
    implementation:
      container:
        image: busybox
        command:
        - sh
        - -c
        args:
        - echo
        - Got scheduled
    """)()


@dsl.pipeline(
    name='echo',
    description='echo pipeline'
)
def echo_pipeline():
    echo_op()


pipeline_conf = kfp_tekton.compiler.pipeline_utils.TektonPipelineConf()
pipeline_conf.add_pipeline_label('test', 'label')
pipeline_conf.add_pipeline_label('test2', 'label2')
pipeline_conf.add_pipeline_annotation('test', 'annotation')
pipeline_conf.set_security_context(V1SecurityContext(run_as_user=0))
pipeline_conf.set_automount_service_account_token(False)
pipeline_conf.add_pipeline_env('WATSON_CRED', 'ABCD1234')
pipeline_conf.add_pipeline_workspace(workspace_name="new-ws", volume=V1Volume(
    name='data',
    persistent_volume_claim=V1PersistentVolumeClaimVolumeSource(
        claim_name='data-volume')
), path_prefix='artifact_data/')
pipeline_conf.add_pipeline_workspace(workspace_name="new-ws-template",
    volume_claim_template_spec=V1PersistentVolumeClaimSpec(
        access_modes=["ReadWriteOnce"],
        resources=V1ResourceRequirements(requests={"storage": "30Gi"})
))


if __name__ == "__main__":
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(echo_pipeline, 'echo_pipeline.yaml', tekton_pipeline_conf=pipeline_conf)
