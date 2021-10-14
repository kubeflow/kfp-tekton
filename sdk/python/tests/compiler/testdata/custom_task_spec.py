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

MY_CUSTOM_TASK_IMAGE_NAME = "veryunique/image:latest"
from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES
TEKTON_CUSTOM_TASK_IMAGES = TEKTON_CUSTOM_TASK_IMAGES.append(MY_CUSTOM_TASK_IMAGE_NAME)

CUSTOM_STR = """
name: any-name
description: custom task
implementation:
  container:
    image: %s
    command:
    - any
    - command
    args:
    - --apiVersion
    - custom_task_api_version
    - --kind
    - custom_task_kind
    - --name
    - custom_task_name
    - --taskSpec
    - |
      {"raw": "raw"}
    - --other_custom_task_argument_keys
    - args
""" % MY_CUSTOM_TASK_IMAGE_NAME

custom_op = components.load_component_from_text(CUSTOM_STR)


@dsl.pipeline(
    name='tekton-custom-task-on-kubeflow-pipeline',
    description='Shows how to use Tekton custom task with custom spec on KFP'
)
def custom_task_pipeline():
    test = custom_op()
    test.add_pod_annotation("valid_container", "false")
    test2 = custom_op().after(test)
    test2.add_pod_annotation("valid_container", "false")


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(custom_task_pipeline, __file__.replace('.py', '.yaml'))
