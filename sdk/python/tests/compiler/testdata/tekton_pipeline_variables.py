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
    name="Tekton Pipeline Variables",
    description="Tests for Tekton Pipeline Variables",
)
def tekton_pipeline_variables(
):
    task1 = dsl.ContainerOp(
        name="task1",
        image="registry.access.redhat.com/ubi8/ubi-minimal",
        command=["/bin/bash", "-c"],
        arguments=["echo \
            Pipeline name: $(context.pipeline.name), \
            PipelineRun name: $(context.pipelineRun.name), \
            PipelineRun namespace: $(context.pipelineRun.namespace), \
            pipelineRun id: $(context.pipelineRun.uid)"
        ]
    )


if __name__ == "__main__":
    TektonCompiler().compile(tekton_pipeline_variables, "tekton_pipeline_variables" + ".yaml")
