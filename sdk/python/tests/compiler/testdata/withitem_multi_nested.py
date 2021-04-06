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
from typing import List


class Coder:
    def empty(self):
        return ""


TektonCompiler._get_unique_id_code = Coder.empty


@dsl.pipeline(name='my-pipeline')
def pipeline(my_pipe_param: list = [100, 200], my_pipe_param3: list = [1, 2]):
    loop_args = [{'a': 1, 'b': 2}, {'a': 10, 'b': 20}]
    with dsl.ParallelFor(loop_args) as item:
        op1 = dsl.ContainerOp(
            name="my-in-coop1",
            image="library/bash:4.4.23",
            command=["sh", "-c"],
            arguments=["echo op1 %s" % (item.a)],
        )

        with dsl.ParallelFor(my_pipe_param) as inner_item:
            op11 = dsl.ContainerOp(
                name="my-inner-inner-coop",
                image="library/bash:4.4.23",
                command=["sh", "-c"],
                arguments=["echo op1 %s %s" % (item.a, inner_item)],
            )
            my_pipe_param2: List[int] = [4, 5]
            with dsl.ParallelFor(my_pipe_param2) as inner_item:
                op12 = dsl.ContainerOp(
                    name="my-inner-inner-coop",
                    image="library/bash:4.4.23",
                    command=["sh", "-c"],
                    arguments=["echo op1 %s %s" % (item.b, inner_item)],
                )
                with dsl.ParallelFor(my_pipe_param3) as inner_item:
                    op13 = dsl.ContainerOp(
                        name="my-inner-inner-coop",
                        image="library/bash:4.4.23",
                        command=["sh", "-c"],
                        arguments=["echo op1 %s %s" % (item.b, inner_item)],
                    )

        op2 = dsl.ContainerOp(
            name="my-in-coop2",
            image="library/bash:4.4.23",
            command=["sh", "-c"],
            arguments=["echo op2 %s" % item.b],
        )


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(pipeline, __file__.replace('.py', '.yaml'))
