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

import itertools

from kfp import dsl, components
from kfp.components import load_component_from_text
from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES, Loop
from kfp_tekton.compiler import TektonCompiler
import yaml


ARTIFACT_FETCHER_IMAGE_NAME = "fetcher/image:latest"
TEKTON_CUSTOM_TASK_IMAGES = TEKTON_CUSTOM_TASK_IMAGES.append(ARTIFACT_FETCHER_IMAGE_NAME)

_artifact_fetcher_no = 0


def artifact_fetcher(**artifact_paths: str):
  '''A containerOp template resolving some artifacts, given their paths.'''
  global _artifact_fetcher_no
  template_yaml = {
    'name': f'artifact-fetcher-{_artifact_fetcher_no}',
    'description': 'Artifact Fetch',
    'inputs': [
      {'name': name, 'type': 'String'}
      for name in artifact_paths.keys()
    ],
    'outputs': [
      {'name': name, 'type': 'Artifact'}
      for name in artifact_paths.keys()
    ],
    'implementation': {
      'container': {
        'image': ARTIFACT_FETCHER_IMAGE_NAME,
        'command': ['sh', '-c'],  # irrelevant
        'args': [
          '--apiVersion', 'fetcher.tekton.dev/v1alpha1',
          '--kind', 'FETCHER',
          '--name', 'artifact_fetcher',
          *itertools.chain(*[
            (f'--{name}', {'inputValue': name})
            for name in artifact_paths.keys()
          ])
        ]
      }
    }
  }
  _artifact_fetcher_no += 1
  template_str = yaml.dump(template_yaml, Dumper=yaml.SafeDumper)
  template = components.load_component_from_text(template_str)
  op = template(**artifact_paths)
  op.add_pod_annotation("valid_container", "false")
  return op


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


@dsl.pipeline("output_in_range_and_pass")
def output_in_range_and_pass():
  op0 = PrintOp('print-0', f"Hello!")
  with Loop.range(1, op0.output, step=2):
    op1 = PrintOp('print-1', f"print {op0.output}")
    op2 = artifact_fetcher(path=op0.output)


if __name__ == '__main__':
  TektonCompiler().compile(output_in_range_and_pass, __file__.replace('.py', '.yaml'))
