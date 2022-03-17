# Copyright 2021 kubeflow.org
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
  op = template(**artifact_paths)
  op.add_pod_annotation("valid_container", "false")
  return op


@dsl.pipeline("literal-params-test")
def literal_params_test(foo: str):
  global _artifact_fetcher_no

  # no literals
  _artifact_fetcher_no = 0
  op00 = artifact_fetcher(bar=foo)
  op01 = artifact_fetcher(foo=foo)

  # not all inputs are literals
  _artifact_fetcher_no = 10
  op10 = artifact_fetcher(foo="foo", bar=foo)
  op11 = artifact_fetcher(foo=foo, bar="bar")

  # all inputs are literals but none of them
  #   matches the name of any pipeline param
  _artifact_fetcher_no = 20
  op20 = artifact_fetcher(bar="bar")
  op21 = artifact_fetcher(bar="bar", buzz="buzz")

  # all inputs are literals and at least one of them
  #   matches the name of a pipeline param
  _artifact_fetcher_no = 30
  op30 = artifact_fetcher(foo="foo")
  op31 = artifact_fetcher(foo="bar")  # doesn't matter what the literal is
  op32 = artifact_fetcher(foo="foo", bar="bar")
  op33 = artifact_fetcher(foo="bar", bar="foo")


if __name__ == '__main__':
  TektonCompiler().compile(literal_params_test, __file__.replace('.py', '.yaml'))
