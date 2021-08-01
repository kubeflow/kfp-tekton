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
from kfp_tekton.compiler import TektonCompiler

TASK_STR_1 = """
name: cache-enabled
description: cached component
outputs:
  - {name: out, type: String}
implementation:
  container:
    image: registry.access.redhat.com/ubi8/ubi-minimal
    command:
    - /bin/bash
    - -c
    - sleep 30 | echo 'hello world' | tee $0
    - {outputPath: out}
"""

TASK_STR_2 = """
name: cache-disabled
description: cached component
outputs:
  - {name: out, type: String}
implementation:
  container:
    image: registry.access.redhat.com/ubi8/ubi-minimal
    command:
    - /bin/bash
    - -c
    - sleep 30 | echo 'hello world' | tee $0
    - {outputPath: out}
"""

task1_op = components.load_component_from_text(TASK_STR_1)
task2_op = components.load_component_from_text(TASK_STR_2)


@dsl.pipeline(
    name="cache",
    description="Example of caching",
)
def cache_pipeline(
):
    task1 = task1_op()
    task2 = task2_op()
    task2.add_pod_label('pipelines.kubeflow.org/cache_enabled', 'false')


if __name__ == "__main__":
    TektonCompiler().compile(cache_pipeline, "cache" + ".yaml")
