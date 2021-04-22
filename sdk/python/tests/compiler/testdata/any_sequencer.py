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
from kfp_tekton.compiler import TektonCompiler
from kfp_tekton.tekton import after_any


def flip_coin_op():
    """Flip a coin and output heads or tails randomly."""
    return dsl.ContainerOp(
        name='flipCoin',
        image='python:alpine3.6',
        command=['sh', '-c'],
        arguments=['python -c "import random; result = \'heads\' if random.randint(0,1) == 0 '
                'else \'tails\'; print(result)" | tee /tmp/output'],
        file_outputs={'output': '/tmp/output'}
    )


@dsl.pipeline(
    name="Any Sequencer",
    description="Any Sequencer Component Demo",
)
def any_sequence_pipeline(
):
    task1 = dsl.ContainerOp(
        name="task1",
        image="registry.access.redhat.com/ubi8/ubi-minimal",
        command=["/bin/bash", "-c"],
        arguments=["sleep 15"]
    )

    task2 = dsl.ContainerOp(
        name="task2",
        image="registry.access.redhat.com/ubi8/ubi-minimal",
        command=["/bin/bash", "-c"],
        arguments=["sleep 200"]
    )

    task3 = dsl.ContainerOp(
        name="task3",
        image="registry.access.redhat.com/ubi8/ubi-minimal",
        command=["/bin/bash", "-c"],
        arguments=["sleep 300"]
    )

    flip_out = flip_coin_op()

    flip_out.after(task1)

    dsl.ContainerOp(
        name="task4",
        image="registry.access.redhat.com/ubi8/ubi-minimal",
        command=["/bin/bash", "-c"],
        arguments=["sleep 30"]
    ).apply(after_any([task2, task3, flip_out.outputs['output'] == "heads"], "any_test"))


if __name__ == "__main__":
    TektonCompiler().compile(any_sequence_pipeline, "any_sequencer" + ".yaml")
