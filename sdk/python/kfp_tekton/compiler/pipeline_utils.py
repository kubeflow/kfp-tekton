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
from kubernetes.client.models import V1SecurityContext, V1Volume, V1PersistentVolumeClaimSpec
from kfp_tekton.compiler._op_to_template import TEKTON_BASH_STEP_IMAGE
from typing import Dict

TEKTON_PIPELINE_ANNOTATIONS = ['sidecar.istio.io/inject', 'tekton.dev/artifact_bucket',
                               'tekton.dev/artifact_endpoint', 'tekton.dev/artifact_endpoint_scheme',
                               'tekton.dev/artifact_items', 'tekton.dev/input_artifacts', 'tekton.dev/output_artifacts']


class TektonPipelineConf(dsl.PipelineConf):
    """PipelineConf contains pipeline level settings."""

    def __init__(self, **kwargs):
        self.pipeline_labels = {}
        self.pipeline_annotations = {}
        self.tekton_inline_spec = True
        self.resource_in_separate_yaml = False
        self.security_context = None
        self.automount_service_account_token = None
        self.pipeline_env = {}
        self.pipeline_workspaces = {}
        self.generate_component_spec_annotations = True
<<<<<<< HEAD
=======
        self.condition_image_name = "python:3.9.17-alpine3.18"
        self.bash_image_name = TEKTON_BASH_STEP_IMAGE
>>>>>>> 972c8817f (feat(sdk): add bash script name config (#1334))
        super().__init__(**kwargs)

    def copy(self):
        return TektonPipelineConf()\
            .set_tekton_inline_spec(self.tekton_inline_spec)\
            .set_resource_in_separate_yaml(self.resource_in_separate_yaml)\
            .set_security_context(self.security_context)\
            .set_automount_service_account_token(self.automount_service_account_token)\
            .set_pipeline_env(self.pipeline_env)

    def add_pipeline_label(self, label_name: str, value: str):
        self.pipeline_labels[label_name] = value
        return self

    def add_pipeline_annotation(self, annotation_name: str, value: str):
        if annotation_name in TEKTON_PIPELINE_ANNOTATIONS:
            raise ValueError('Cannot add pipeline annotation %s:%s because it is a reserved Tekton annotation.'
                             % annotation_name, value)
        self.pipeline_annotations[annotation_name] = value
        return self

    def set_tekton_inline_spec(self, value: bool):
        self.tekton_inline_spec = value
        return self

    def set_resource_in_separate_yaml(self, value: bool):
        self.resource_in_separate_yaml = value
        return self

    def set_security_context(self, value: V1SecurityContext):
        self.security_context = value
        return self

    def set_automount_service_account_token(self, value: bool):
        self.automount_service_account_token = value
        return self

    def add_pipeline_env(self, env_name: str, value: str):
        self.pipeline_env[env_name] = value
        return self

    def set_pipeline_env(self, value: Dict[str, str]):
        self.pipeline_env = value
        return self

    def add_pipeline_workspace(self,
                               workspace_name: str,
                               volume: V1Volume = None,
                               volume_claim_template_spec: V1PersistentVolumeClaimSpec = None,
                               path_prefix: str = None):
        if not (volume or volume_claim_template_spec) or (volume and volume_claim_template_spec):
            raise ValueError("Only either volume or volume_claim_templateneeds to be defined in add_pipeline_workspace(%s)" %
                             workspace_name)
        volumeSpec = volume if volume else volume_claim_template_spec
        self.pipeline_workspaces[workspace_name] = (volumeSpec, path_prefix)
        return self

    def set_generate_component_spec_annotations(self, value: bool):
        self.generate_component_spec_annotations = value
        return self
<<<<<<< HEAD
=======

    def set_condition_image_name(self, condition_image_name: str):
        self.condition_image_name = condition_image_name
        return self

    def set_bash_image_name(self, bash_image_name: str):
        self.bash_image_name = bash_image_name
        return self
>>>>>>> 972c8817f (feat(sdk): add bash script name config (#1334))
