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


from attr import dataclass, asdict, fields

from kfp import dsl
from kfp.components import load_component_from_text
from kfp_tekton.compiler import TektonCompiler
from kfp_tekton.tekton import AddOnGroup

CEL_TASK_IMAGE_NAME = "veryunique/image:latest"
from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES
TEKTON_CUSTOM_TASK_IMAGES = TEKTON_CUSTOM_TASK_IMAGES.append(CEL_TASK_IMAGE_NAME)


@dataclass
class ExitHandlerFields:
    status: dsl.PipelineParam
    status_message: dsl.PipelineParam

    def __getitem__(self, item: str) -> dsl.PipelineParam:
        di = asdict(self)
        return di.get(item) or di.get(item.replace('-', '_'))


class ExitHandler(AddOnGroup):
    """A custom OpsGroup which maps to a custom task"""
    def __init__(self):
        labels = {
            'pipelines.kubeflow.org/cache_enabled': 'false',
        }
        annotations = {
            'ws-pipelines.ibm.com/pipeline-cache-enabled': 'false',
        }

        super().__init__(
            kind='Exception',
            api_version='custom.tekton.dev/v1alpha1',
            params={},
            is_finally=True,
            labels=labels,
            annotations=annotations,
        )

        internal_params = {
            field.name: AddOnGroup.create_internal_param(field.name)
            for field in fields(ExitHandlerFields)
        }
        self._internal_params = internal_params
        self.fields = ExitHandlerFields(
            status=internal_params['status'],
            status_message=internal_params['status_message'],
        )

    def __enter__(self) -> ExitHandlerFields:
        super().__enter__()
        return self.fields

    def post_params(self, params: list) -> list:
        params_map = {
            param['name']: param for param in params
        }
        internal_params_names = set(self._internal_params.keys())
        params_map = {
            k: v for k, v in params_map.items()
            if k not in internal_params_names
        }
        params = list(params_map.values())
        params.append({
            'name': 'pipelinerun_name',
            'value': '$(context.pipelineRun.name)',
        })
        return params

    def post_task_spec(self, task_spec: dict) -> dict:
        spec = task_spec.get('spec') or {}

        pod_template = spec.setdefault('podTemplate', {})
        pod_template['imagePullSecrets'] = [{'name': 'ai-lifecycle'}]
        pod_template['automountServiceAccountToken'] = 'false'

        pipeline_spec = spec.get('pipelineSpec') or {}
        params = pipeline_spec.get('params') or []
        params_map = {
            param['name']: param for param in params
        }
        params_map.update({
            k: {
                'name': k,
                'type': 'string',
            } for k in self._internal_params.keys()
        })
        params = list(params_map.values())
        spec = task_spec.setdefault('spec', {})
        spec.setdefault('pipelineSpec', {})['params'] = params
        task_spec['spec'] = spec
        return task_spec


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


@dsl.pipeline("test pipeline")
def test_pipeline():
    with ExitHandler() as it:
        # PrintOp("print-err-status", it.status)
        cel = load_component_from_text(r"""
            name: cel
            inputs:
            - {name: cel-input}
            outputs:
            - {name: cel-output}
            implementation:
              container:
                image: veryunique/image:latest
                command: [cel]
                args:
                - --apiVersion
                - custom.tekton.dev/v1alpha1
                - --kind
                - Cel
                - --name
                - cel_123
                - --status
                - {inputValue: cel-input}
                - --taskSpec
                - '{}'
        """)(it.status)
        cel.add_pod_annotation("valid_container", "false")

    PrintOp("print", "some-message")


if __name__ == '__main__':
  TektonCompiler().compile(test_pipeline, __file__.replace('.py', '.yaml'))
