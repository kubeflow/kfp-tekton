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

import kfp.dsl as dsl
from kfp_tekton.compiler import TektonCompiler


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty


@dsl.pipeline(name='my-pipeline')
def pipeline(loopidy_doop: list = [3, 5, 7, 9]):
    op0 = dsl.ContainerOp(
        name="my-out-cop0",
        image='python:alpine3.6',
        command=["sh", "-c"],
        arguments=[
            'python -c "import json; import sys; json.dump([i for i in range(20, 31)], open(\'/tmp/out.json\', \'w\'))"'],
        file_outputs={'out': '/tmp/out.json'},
    )

    with dsl.ParallelFor(loopidy_doop) as item:
        op1 = dsl.ContainerOp(
            name="my-in-cop1",
            image="library/bash:4.4.23",
            command=["sh", "-c"],
            arguments=["echo no output global op1, item: %s" % item],
        ).after(op0)

    op_out = dsl.ContainerOp(
        name="my-out-cop2",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo no output global op2, outp: %s" % op0.output],
    ).after(op1)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
