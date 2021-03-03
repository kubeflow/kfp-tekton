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


@dsl.pipeline(name='my-pipeline')
def pipeline(my_pipe_param: int = 10):
    loop_args = [1, 2]
    with dsl.ParallelFor(loop_args) as item:
        op1 = dsl.ContainerOp(
            name="my-in-coop1",
            image="library/bash:4.4.23",
            command=["sh", "-c"],
            arguments=["echo op1 %s %s" % (item, my_pipe_param)],
        )

        with dsl.ParallelFor([100, 200, 300]) as inner_item:
            op11 = dsl.ContainerOp(
                name="my-inner-inner-coop",
                image="library/bash:4.4.23",
                command=["sh", "-c"],
                arguments=["echo op1 %s %s %s" % (item, inner_item, my_pipe_param)],
            )

        op2 = dsl.ContainerOp(
            name="my-in-coop2",
            image="library/bash:4.4.23",
            command=["sh", "-c"],
            arguments=["echo op2 %s" % item],
        )

    op_out = dsl.ContainerOp(
        name="my-out-cop",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo %s" % my_pipe_param],
    )


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
