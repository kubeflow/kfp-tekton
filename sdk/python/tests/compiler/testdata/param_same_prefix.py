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
from typing import Mapping
import yaml

from kfp import dsl, components
from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES, Loop
from kfp_tekton.compiler import TektonCompiler


ARTIFACT_FETCHER_IMAGE_NAME = "fetcher/image:latest"
TEKTON_CUSTOM_TASK_IMAGES = TEKTON_CUSTOM_TASK_IMAGES.append(ARTIFACT_FETCHER_IMAGE_NAME)


class Coder:
  def empty(self):
    return ""


TektonCompiler._get_unique_id_code = Coder.empty


def fetcher_op(name: str, artifact_paths: Mapping[str, str]):
  template_yaml = {
    'name': name,
    'description': 'Fetch',
    'inputs': [
      {'name': key, 'type': 'String'}
      for key in artifact_paths.keys()
    ],
    'outputs': [
      {'name': key, 'type': 'String'}
      for key in artifact_paths.keys()
    ],
    'implementation': {
      'container': {
        'image': ARTIFACT_FETCHER_IMAGE_NAME,
        'command': ['sh', '-c'],  # irrelevant
        'args': [
          '--apiVersion', 'fetcher.tekton.dev/v1alpha1',
          '--kind', 'FETCHER',
          '--name', 'fetcher_op',
          *itertools.chain(*[
            (f'--{key}', {'inputValue': key})
            for key in artifact_paths.keys()
          ])
        ]
      }
    }
  }
  template_str = yaml.dump(template_yaml, Dumper=yaml.SafeDumper)
  template = components.load_component_from_text(template_str)
  args = {
    key.replace('-', '_'): val
    for key, val in artifact_paths.items()
  }
  op = template(**args)
  op.add_pod_annotation("valid_container", "false")
  return op


def print_op(name: str, messages: Mapping[str, str]):
  inputs = '\n'.join([
    '  - {name: %s, type: String}' % key for key in messages.keys()
  ])
  outputs = '\n'.join([
    '  - {name: %s, type: String}' % key for key in messages.keys()
  ])
  inout_refs = '\n'.join(list(itertools.chain(*[
    (
      '      - {inputValue: %s}' % key,
      '      - {outputPath: %s}' % key,
    ) for key in messages.keys()
  ])))
  print_template_str = """
  name: %s
  inputs:
%s
  outputs:
%s
  implementation:
    container:
      image: alpine:3.6
      command:
      - sh
      - -c
      - ...
%s
  """ % (name, inputs, outputs, inout_refs)
  print_template = components.load_component_from_text(
    print_template_str
  )
  args = {
    key.replace('-', '_'): val
    for key, val in messages.items()
  }
  return print_template(**args)


def string_consumer(name: str, msg: str = None):
  if msg is None:
    msg = name
  template = components.load_component_from_text(
  """
  name: %s
  inputs:
  - {name: input_text, type: String}
  outputs:
  - {name: output_value, type: String}
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
  return template(msg)


@dsl.pipeline("prefixes")
def prefixes(foo: str, li: list = [1, 2, 3]):

  # custom-task, prefix diff from name
  fetch_00 = fetcher_op('foo-00', {'bar-00-val': foo})

  # custom-task, prefix same as name
  fetch_01 = fetcher_op('foo-01', {'foo-01-val': foo})

  # custom-task, param name identical to name
  fetch_02 = fetcher_op('foo-02', {'foo-02': foo})

  # normal task, prefix diff from name
  print_10 = print_op('foo-10', {'bar-10-val': foo})

  # normal task, prefix same as name
  print_11 = print_op('foo-11', {'foo-11-val': foo})

  # normal task, param name identical to name
  print_12 = print_op('foo-12', {'foo-12': foo})

  with Loop(li):
    string_consumer('fetch-00', fetch_00.output)
    string_consumer('fetch-01', fetch_01.output)
    string_consumer('fetch-02', fetch_02.output)
    string_consumer('print-10', print_10.output)
    string_consumer('print-11', print_11.output)
    string_consumer('print-12', print_12.output)


if __name__ == '__main__':
  TektonCompiler().compile(prefixes, __file__.replace('.py', '.yaml'))
