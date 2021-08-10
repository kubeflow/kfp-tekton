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

from kfp import dsl, components


def print_op(msg: str):
    """Print a message."""
    return components.load_component_from_text("""
    name: print
    description: print out msg
    inputs:
      - {name: msg, type: String}
    implementation:
      container:
        image: alpine:3.6
        command:
        - echo
        - {inputValue: msg}
    """)(msg=msg)


def add_annotation_and_label(op):
    op.add_pod_annotation(name='hobby', value='football')
    op.add_pod_label(name='hobby', value='football')
    return op


@dsl.pipeline(
    name='pipeline-transformer',
    description='The pipeline shows how to apply functions to all ops in the pipeline by pipeline transformers'
)
def transform_pipeline():
    op1 = print_op('hey, what are you up to?')
    op2 = print_op('train my model.')
    dsl.get_pipeline_conf().add_op_transformer(add_annotation_and_label)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(transform_pipeline, __file__.replace('.py', '.yaml'))
