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
from kfp_tekton.tekton import AddOnGroup, AnySequencer


class ExitHandler(AddOnGroup):
    """A custom OpsGroup which maps to a custom task"""
    def __init__(self):
        labels = {
            'pipelines.kubeflow.org/cache_enabled': 'false',
        }

        super().__init__(
            kind='Exception',
            api_version='custom.tekton.dev/v1alpha1',
            params={},
            is_finally=True,
            labels=labels,
            annotations={},
        )

    def __enter__(self):
        return super().__enter__()


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
  """ % name
  )
  return print_op(msg)


@dsl.pipeline("any-sequencer in finally")
def any_sequencer_in_finally():
  print00 = PrintOp("print-00")
  print01 = PrintOp("print-01")
  AnySequencer([print00, print01])

  with ExitHandler():
      print10 = PrintOp("print-10")
      print11 = PrintOp("print-11")
      AnySequencer([print10, print11])


if __name__ == '__main__':
  TektonCompiler().compile(any_sequencer_in_finally, __file__.replace('.py', '.yaml'))
