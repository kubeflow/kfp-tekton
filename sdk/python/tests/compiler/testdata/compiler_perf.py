# Copyright 2023 kubeflow.org
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
from kfp.components import load_component_from_text


argument_count = 2
yaml_text = """
name: echo
inputs:%s
implementation:
  container:
    image: busybox
    command:
    - sh
    - -c
    args:
    - echo%s

""" % (("".join(["\n  - {name: hello%s, type: String}" % i for i in range(argument_count)])),
       ("".join(["\n    - {inputValue: hello%s}" % i for i in range(argument_count)])))


def return_n_args(arg, n):
    return (arg,) * n


@dsl.pipeline(
    name='echo',
    description='echo pipeline'
)
def echo_pipeline(name: str = 'latest'):
    n_args = return_n_args(name, argument_count)
    for i in range(1500):
        echo_op = load_component_from_text(yaml_text)(*n_args)


if __name__ == "__main__":
    from kfp_tekton.compiler import TektonCompiler
    import time
    start_time = time.time()
    TektonCompiler().compile(echo_pipeline, 'echo_pipeline.yaml')
    end_time = time.time()
    print("time diff %s" % (end_time - start_time))
