# TODO: from KFP 1.3.0, need to implement for kfp_tekton.compiler

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


@dsl.graph_component
def echo1_graph_component(text1: str):
  components.load_component_from_text("""
  name: echo1-task1
  description: echo task
  inputs:
    - {name: text, type: String}
  implementation:
    container:
      image: library/bash:4.4.23
      command:
      - sh
      - -c
      args:
      - echo
      - {inputValue: text}
  """)(text=text1)


@dsl.graph_component
def echo2_graph_component(text2: str):
  components.load_component_from_text("""
  name: echo2-task1
  description: echo task
  inputs:
    - {name: text, type: String}
  implementation:
    container:
      image: library/bash:4.4.23
      command:
      - sh
      - -c
      args:
      - echo
      - {inputValue: text}
  """)(text=text2)


@dsl.pipeline(name='opsgroups-pipeline')
def opsgroups_pipeline(text1: str = 'message 1', text2: str = 'message 2'):
  step1_graph_component = echo1_graph_component(text1)
  step2_graph_component = echo2_graph_component(text2)
  step2_graph_component.after(step1_graph_component)


if __name__ == '__main__':
  from kfp_tekton.compiler import TektonCompiler
  TektonCompiler().compile(opsgroups_pipeline, __file__.replace('.py', '.yaml'))
