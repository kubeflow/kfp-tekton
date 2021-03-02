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


class FlipCoinOp(dsl.ContainerOp):

    def __init__(self, name, forced_result=""):
        super(FlipCoinOp, self).__init__(
            name=name,
            image='python:alpine3.6',
            command=['sh', '-c'],
            arguments=['python -c "import random; import sys; forced_result = \''+forced_result+'\'; '
                       'result = \'heads\' if random.randint(0,1) == 0 else \'tails\'; '
                       'print(forced_result) if (forced_result == \'heads\' or forced_result == \'tails\') else print(result)"'
                       ' | tee /tmp/output'],
            file_outputs={'output': '/tmp/output'})


class PrintOp(dsl.ContainerOp):

    def __init__(self, name, msg):
        super(PrintOp, self).__init__(
            name=name,
            image='alpine:3.6',
            command=['echo', msg])


@dsl.pipeline(
    name='Flip Coin Example Pipeline',
    description='Shows how to use dsl.Condition.'
)
def flipcoin(forced_result1: str = 'heads', forced_result2: str = 'tails'):
    flip = FlipCoinOp('flip', str(forced_result1))

    with dsl.Condition(flip.output == 'heads') as condition:
        flip2 = FlipCoinOp('flip-again', str(forced_result2))

        with dsl.Condition(flip2.output == 'tails'):
            PrintOp('print1', flip2.output)

    with dsl.Condition(flip.output == 'tails') as condition_2:
        PrintOp('print2', flip.output)

    PrintOp('print3', 'done').after(condition).after(condition_2)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(flipcoin, __file__.replace('.py', '.yaml'))

