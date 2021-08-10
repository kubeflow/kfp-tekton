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


def random_failure_op(exit_codes: str):
    """A component that fails randomly."""
    return components.load_component_from_text("""
    name: random-failure
    description: random failure
    inputs:
      - {name: exitcode, type: String}
    implementation:
      container:
        image: python:alpine3.6
        command:
        - python
        - -c
        args:
        - |
          import random; import sys; exit_code = random.choice([int(i) for i in sys.argv[1].split(",")]); \
          print(exit_code); sys.exit(exit_code)
        - {inputValue: exitcode}
    """)(exitcode=exit_codes)


@dsl.pipeline(
    name='retry-random-failures',
    description='The pipeline includes two steps which fail randomly. It shows how to use ContainerOp(...).set_retry(...).'
)
def retry_sample_pipeline():
    op1 = random_failure_op('0,1,2,3').set_retry(10)
    op2 = random_failure_op('0,1').set_retry(5)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(retry_sample_pipeline, __file__.replace('.py', '.yaml'))
