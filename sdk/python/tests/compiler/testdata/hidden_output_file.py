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

from kfp import dsl


def write_text_op():
    return dsl.ContainerOp(
        name='Download file',
        image='aipipeline/echo-text:latest',
        command=['/bin/bash'],
        arguments=['-c', '/echo.sh'],
        file_outputs={
            'data': '/tmp/results.txt',
        }
    )


def echo2_op(text1):
    return dsl.ContainerOp(
        name='echo',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        arguments=['echo "Text 1: $0";', text1]
    )


@dsl.pipeline(
  name='Hidden output file pipeline',
  description='Run a script that passes file to a non configurable path'
)
def hidden_output_file_pipeline(
):
    """A three-step pipeline with the first two steps running in parallel."""

    write_text = write_text_op()

    echo_task = echo2_op(write_text.output)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(hidden_output_file_pipeline, __file__.replace('.py', '.yaml'))
