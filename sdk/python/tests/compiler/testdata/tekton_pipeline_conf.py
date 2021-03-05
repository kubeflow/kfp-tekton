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
import kfp_tekton


def echo_op():
    return dsl.ContainerOp(
        name='echo',
        image='busybox',
        command=['sh', '-c'],
        arguments=['echo "Got scheduled"']
    )


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


if __name__ == "__main__":
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(echo_pipeline, 'echo_pipeline.yaml', tekton_pipeline_conf=pipeline_conf)
