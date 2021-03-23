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
from kfp_tekton.compiler import TektonCompiler


@dsl.pipeline(
    name="Cache",
    description="Example of caching",
)
def cache_pipeline(
):
    task1 = dsl.ContainerOp(
        name="cache-enabled",
        image="registry.access.redhat.com/ubi8/ubi-minimal",
        command=["/bin/bash", "-c"],
        arguments=["sleep 30 | echo 'hello world' | tee /tmp/output"],
        file_outputs={'output_value': '/tmp/output'}
    )
    task2 = dsl.ContainerOp(
        name="cache-disabled ",
        image="registry.access.redhat.com/ubi8/ubi-minimal",
        command=["/bin/bash", "-c"],
        arguments=["sleep 30 | echo 'hello world' | tee /tmp/output"],
        file_outputs={'output_value': '/tmp/output'}
    )
    task2.add_pod_label('pipelines.kubeflow.org/cache_enabled', 'false')


if __name__ == "__main__":
    TektonCompiler().compile(cache_pipeline, "cache" + ".yaml")
