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
from kfp_tekton.tekton import CEL_ConditionOp
from kfp import components


def random_num(low:int, high:int) -> int:
    """Generate a random number between low and high."""
    import random
    result = random.randint(low, high)
    print(result)
    return result

def flip_coin() -> str:
    """Flip a coin and output heads or tails randomly."""
    import random
    result = 'heads' if random.randint(0, 1) == 0 else 'tails'
    print(result)
    return result

def print_msg(msg: str):
    """Print a message."""
    print(msg)


flip_coin_op = components.create_component_from_func(
    flip_coin, base_image='python:alpine3.6')
print_op = components.create_component_from_func(
    print_msg, base_image='python:alpine3.6')
random_num_op = components.create_component_from_func(
    random_num, base_image='python:alpine3.6')


@dsl.pipeline(
    name='conditional-execution-pipeline',
    description='Shows how to use dsl.Condition() and task dependencies on multiple condition branches.'
)
def flipcoin_pipeline():
    flip = flip_coin_op()
    cel_condition = CEL_ConditionOp("'%s' != 'hey'" % flip.output)
    with dsl.Condition(cel_condition.output == 'true'):
        random_num_head = random_num_op(6, 9)
        cel_condition_2 = CEL_ConditionOp("%s > 5" % random_num_head.output)
        with dsl.Condition(cel_condition_2.output == 'true'):
            print_op('heads and %s > 5!' % random_num_head.output)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    import kfp_tekton
    pipeline_conf = kfp_tekton.compiler.pipeline_utils.TektonPipelineConf()
    pipeline_conf.add_pipeline_label('pipelines.kubeflow.org/cache_enabled', 'false')
    TektonCompiler().compile(flipcoin_pipeline, __file__.replace('.py', '.yaml'), tekton_pipeline_conf=pipeline_conf)
