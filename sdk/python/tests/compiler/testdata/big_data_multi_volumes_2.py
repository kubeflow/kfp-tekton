# Copyright 2022 kubeflow.org
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
from kfp_tekton.compiler import TektonCompiler
from kubernetes.client import V1Volume, V1VolumeMount, V1SecretVolumeSource


def PrintOp(name: str, msg0: str = None, msg1: str = None):
  if msg0 is None:
    msg0 = name
  if msg1 is None:
    msg1 = msg0
  print_op = load_component_from_text(
  """
  name: %s
  inputs:
  - {name: input_text_0, type: String, description: 'Represents an input parameter.'}
  - {name: input_text_1, type: String, description: 'Represents an input parameter.'}
  outputs:
  - {name: output_value, type: String, description: 'Represents an output parameter.'}
  implementation:
    container:
      image: alpine:3.6
      command:
      - sh
      - -c
      - |
        set -e
        echo $0 >> $2
        echo $1 >> $2
      - {inputValue: input_text_0}
      - {inputValue: input_text_1}
      - {outputPath: output_value}
  """ % name
  )
  return print_op(msg0, msg1)


def PrintRefOp(name: str, msg0: str = None, msg1: str = None):
  if msg0 is None:
    msg0 = name
  if msg1 is None:
    msg1 = msg0
  print_op = load_component_from_text(
  """
  name: %s
  inputs:
  - {name: input_text_0, type: String, description: 'Represents an input parameter.'}
  - {name: input_text_1, type: String, description: 'Represents an input parameter.'}
  outputs:
  - {name: output_value, type: String, description: 'Represents an output parameter.'}
  implementation:
    container:
      image: alpine:3.6
      command:
      - sh
      - -c
      - |
        set -e
        cat $0 >> $2
        cat $1 >> $2
      - {inputPath: input_text_0}
      - {inputPath: input_text_1}
      - {outputPath: output_value}
  """ % name
  )
  return print_op(msg0, msg1)


def add_volume_and_mount(op: dsl.ContainerOp) -> dsl.ContainerOp:
    suffix = op.name[op.name.index('-'):]
    return op.add_volume(
        V1Volume(
            name='volume' + suffix,
            secret=V1SecretVolumeSource(secret_name='secret' + suffix),
        )
    ).container.add_volume_mount(
        V1VolumeMount(
            name='volume' + suffix,
            mount_path='/volume' + suffix,
        )
    )


@dsl.pipeline(name='big-data')
def big_data():
    # literal -> small
    PrintOp(
        'print-sm',
        'literal0',
        'literal1',
    ).apply(add_volume_and_mount)

    # literal -> big
    PrintRefOp(
        'print-big',
        'literal0',
        'literal1',
    ).apply(add_volume_and_mount)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(big_data, __file__.replace('.py', '.yaml'))
