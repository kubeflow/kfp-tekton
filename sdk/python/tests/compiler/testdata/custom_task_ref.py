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

MY_CUSTOM_TASK_IMAGE_NAME = "veryunique/image:latest"
from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES
TEKTON_CUSTOM_TASK_IMAGES = TEKTON_CUSTOM_TASK_IMAGES.append(MY_CUSTOM_TASK_IMAGE_NAME)


def getCustomOp():
    CustomOp = dsl.ContainerOp(
        name="any-name",
        image=MY_CUSTOM_TASK_IMAGE_NAME,
        command=["any", "command"],
        arguments=["--apiVersion", "custom_task_api_version",
                    "--kind", "custom_task_kind",
                    "--name", "custom_task_name",
                    "--taskRef", {"raw": "raw"},
                    "--other_custom_task_argument_keys", "args"],
        file_outputs={"other_custom_task_argument_keys": '/anypath'}
    )
    # Annotation to tell the Argo controller that this CustomOp is for specific Tekton runtime only.
    CustomOp.add_pod_annotation("valid_container", "false")
    return CustomOp


@dsl.pipeline(
    name='Tekton custom task on Kubeflow Pipeline',
    description='Shows how to use Tekton custom task with custom spec on KFP'
)
def custom_task_pipeline():
    test = getCustomOp()
    test2 = getCustomOp().after(test)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(custom_task_pipeline, __file__.replace('.py', '.yaml'))

