# Copyright 2019-2020 kubeflow.org.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import json

from ._op_to_template import _op_to_template
from kfp.compiler.compiler import Compiler


class TektonCompiler(Compiler) :
  """DSL Compiler to generate Tekton YAML.

  It compiles DSL pipeline functions into workflow yaml. Example usage:
  ```python
  @dsl.pipeline(
    name='name',
    description='description'
  )
  def my_pipeline(a: int = 1, b: str = "default value"):
    ...

  TektonCompiler().compile(my_pipeline, 'path/to/workflow.yaml')
  ```
  """

  def _create_dag_templates(self, pipeline, op_transformers=None, params=None, op_to_templates_handler=None):
    """Create all groups and ops templates in the pipeline.

    Args:
      pipeline: Pipeline context object to get all the pipeline data from.
      op_transformers: A list of functions that are applied to all ContainerOp instances that are being processed.
      op_to_templates_handler: Handler which converts a base op into a list of argo templates.
    """
    op_to_steps_handler = op_to_templates_handler or (lambda op : [_op_to_template(op)])
    root_group = pipeline.groups[0]

    # Call the transformation functions before determining the inputs/outputs, otherwise
    # the user would not be able to use pipeline parameters in the container definition
    # (for example as pod labels) - the generated template is invalid.
    for op in pipeline.ops.values():
      for transformer in op_transformers or []:
        transformer(op)
    steps = []
    for op in pipeline.ops.values():
      steps.extend(op_to_steps_handler(op))

    return steps

  def _create_pipeline_workflow(self, args, pipeline, op_transformers=None, pipeline_conf=None):
    """Create workflow for the pipeline."""
    # Input Parameters
    params = []
    for arg in args:
      param = {'name': arg.name}
      if arg.value is not None:
        if isinstance(arg.value, (list, tuple)):
          param['default'] = json.dumps(arg.value, sort_keys=True)
        else:
          param['default'] = str(arg.value)
      params.append(param)

    # Steps
    steps = self._create_dag_templates(pipeline, op_transformers, params)
    # The whole pipeline workflow
    pipeline_name = pipeline.name or 'Pipeline'

    workflow = {
      'apiVersion': 'tekton.dev/v1alpha1',
      'kind': 'Task',
      'metadata': {'name': pipeline_name},
      'spec': {
        'inputs': {
          'params': params
        },
        'steps': steps
      }
    }

    return workflow
