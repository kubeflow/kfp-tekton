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
from kfp_tekton.compiler import TektonCompiler as Compiler


def gcs_download_op(url):
    return dsl.ContainerOp(
        name='GCS - Download' * 4,
        image='google/cloud-sdk:279.0.0',
        command=['sh', '-c'],
        arguments=['gsutil cat $0 | tee $1', url, '/tmp/results.txt'],
        file_outputs={
            'data' * 10: '/tmp/results.txt',
        }
    )


class PrintOp(dsl.ContainerOp):
    def __init__(self, name, msg):
        super(PrintOp, self).__init__(
            name=name,
            image='alpine:3.6',
            command=['echo', msg])


@dsl.pipeline(name="Some very long name with lots of words in it. " +
                   "It should be over 63 chars long in order to observe the problem.")
def main_fn(url1='gs://ml-pipeline-playground/shakespeare1.txt'):
    download1_task = gcs_download_op(url1)
    PrintOp('print' * 10, download1_task.output)


if __name__ == '__main__':
    Compiler().compile(main_fn, __file__.replace('.py', '.yaml'))
