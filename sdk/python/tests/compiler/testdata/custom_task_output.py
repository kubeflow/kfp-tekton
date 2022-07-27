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
from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES
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
  op = template(**{
    k.lower(): v for k, v in artifact_paths.items()
  })
  op.add_pod_annotation("valid_container", "false")
  return op


_artifact_printer_no = 0


def artifact_printer(artifact):
  global _artifact_printer_no
  template_yaml = {
    'name': f'artifact-printer-{_artifact_printer_no}',
    'description': 'Artifact Fetch',
    'inputs': [
      {'name': 'artifact', 'type': 'Artifact'}
    ],
    'outputs': [
      {'name': 'stdout', 'type': 'String'}
    ],
    'implementation': {
      'container': {
        'image': 'alpine:3.6',
        'command': [
          'echo',
          {'inputValue': 'artifact'},
          '>',
          {'outputPath': 'stdout'}
        ]
      }
    }
  }
  _artifact_printer_no += 1
  template_str = yaml.dump(template_yaml, Dumper=yaml.SafeDumper)
  template = components.load_component_from_text(template_str)
  op = template(artifact)
  return op


@dsl.pipeline("uppercase vs lowercase")
def uppercase_vs_lowercase():
  fetch_upper = artifact_fetcher(FOO='FOO')
  fetch_lower = artifact_fetcher(foo='foo')
  print_upper = artifact_printer(fetch_upper.output)
  print_lower = artifact_printer(fetch_lower.output)


if __name__ == '__main__':
  TektonCompiler().compile(uppercase_vs_lowercase, __file__.replace('.py', '.yaml'))
