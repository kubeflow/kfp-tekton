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
        super().__init__(**kwargs)

    def copy(self):
        return TektonPipelineConf()\
            .add_pipeline_label(self.pipeline_labels)\
            .add_pipeline_annotation(self.pipeline_annotations)\
            .set_tekton_inline_spec(self.tekton_inline_spec)\
            .set_resource_in_separate_yaml(self.resource_in_separate_yaml)

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
