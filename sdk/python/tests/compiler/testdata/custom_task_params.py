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

from typing import Any

from kfp import dsl

custom_task_name = "some-custom-task"
custom_task_api_version = "custom.tekton.dev/v1alpha1"
custom_task_image = "some-image"
custom_task_kind = "custom-task"


def custom_task_args(foo: str, bar: Any, pi: float) -> list:
    return [
        "--foo",
        foo,
        "--bar",
        bar,
        "--pi",
        pi,
    ]


custom_task_results = ["target"]
custom_task_resource_name = "some-custom-resource"
custom_task_resource = {
    "params": [
        {
            "name": "foo",
            "type": "string",
        },
        {
            "name": "bar",
        },
        {
            "name": "pi",
            "type": "int",
            "default": 3
        }
    ]
}

from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES
TEKTON_CUSTOM_TASK_IMAGES = TEKTON_CUSTOM_TASK_IMAGES.append(custom_task_image)


def custom_task(resource_label: str, foo: str, bar: Any, pi: float) -> dsl.ContainerOp:
    task = dsl.ContainerOp(
        name=custom_task_name,
        image=custom_task_image,
        command=["--name", foo],
        arguments=[
            "--apiVersion", custom_task_api_version,
            "--kind", custom_task_kind,
            "--name", custom_task_resource_name,
            *custom_task_args(foo, bar, pi),
            resource_label, custom_task_resource
        ],
        file_outputs={
            f"{result}": f"/{result}"
            for result in custom_task_results
        }
    )
    task.add_pod_annotation("valid_container", "false")
    return task


def main_task_ref(foo: str = "Foo", bar="buzz", pi: int = 3.14):
    custom_task("--taskRef", foo, bar, pi)


def main_task_spec(foo: str = "Foo", bar="buzz", pi: int = 3.14):
    custom_task("--taskSpec", foo, bar, pi)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler as Compiler
  Compiler().compile(main_task_ref, __file__.replace('.py', '_ref.yaml'))
  Compiler().compile(main_task_spec, __file__.replace('.py', '_spec.yaml'))
