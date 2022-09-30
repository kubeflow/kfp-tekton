from kfp import dsl
from kfp.components import load_component_from_text
from kfp_tekton.tekton import Loop
from kfp_tekton.compiler import TektonCompiler
import copy


class Coder:
  def empty(self):
    return ""


TektonCompiler._get_unique_id_code = Coder.empty


def PrintOp(name: str, msg: str = None):
  if msg is None:
    msg = name
  print_op = load_component_from_text(
  """
  name: %s
  inputs:
  - {name: input_text, type: String, description: 'Represents an input parameter.'}
  outputs:
  - {name: output_value, type: String, description: 'Represents an output paramter.'}
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
  """ % (name)
  )
  return print_op(msg)


@dsl.pipeline("loop-multi")
def loop_multi(param: list = ["a", "b", "c"]):
  with Loop(param) as it0:
    with Loop(param) as it1:
      PrintOp("print-01", f"print {it0} {it1}")


if __name__ == '__main__':
  TektonCompiler().compile(loop_multi, __file__.replace('.py', '.yaml'))
