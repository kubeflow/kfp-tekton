from kfp import dsl
from kfp.components import load_component_from_text
from kfp_tekton.compiler import TektonCompiler
from kubernetes.client import V1Volume, V1VolumeMount, V1SecretVolumeSource


def PrintOp(name: str, msg: str = None):
  if msg is None:
    msg = name
  print_op = load_component_from_text(
  """
  name: %s
  inputs:
  - {name: input_text, type: String, description: 'Represents an input parameter.'}
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
        echo $0 > $1
      - {inputValue: input_text}
      - {outputPath: output_value}
  """ % name
  )
  return print_op(msg)

def PrintRefOp(name: str, msg: str = None):
  if msg is None:
    msg = name
  print_op = load_component_from_text(
  """
  name: %s
  inputs:
  - {name: input_text, type: String, description: 'Represents an input parameter.'}
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
        cat $0 > $1
      - {inputPath: input_text}
      - {outputPath: output_value}
  """ % name
  )
  return print_op(msg)

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
        'literal',
    ).apply(add_volume_and_mount)

    # literal -> big
    PrintRefOp(
        'print-big',
        'literal',
    ).apply(add_volume_and_mount)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(big_data, __file__.replace('.py', '.yaml'))