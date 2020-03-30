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
import inspect
import json
import tarfile
import yaml
import zipfile
from typing import Callable, Set, List, Text, Dict, Tuple, Any, Union, Optional

from ._op_to_template import _op_to_template, literal_str

from kfp import dsl
from kfp.compiler._default_transformers import add_pod_env
from kfp.compiler._k8s_helper import sanitize_k8s_name
from kfp.compiler.compiler import Compiler
from kfp.components.structures import InputSpec
from kfp.dsl._metadata import _extract_pipeline_metadata

from .. import tekton_api_version


# pyyaml representer for literal yaml string dumper
def _literal_str_representer(dumper, data):
    return dumper.represent_scalar(u'tag:yaml.org,2002:str', data, style='|')


yaml.add_representer(literal_str, _literal_str_representer)


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
    op_to_steps_handler = op_to_templates_handler or (lambda op: [_op_to_template(op)])
    root_group = pipeline.groups[0]

    # Call the transformation functions before determining the inputs/outputs, otherwise
    # the user would not be able to use pipeline parameters in the container definition
    # (for example as pod labels) - the generated template is invalid.
    for op in pipeline.ops.values():
      for transformer in op_transformers or []:
        transformer(op)
    tasks = []
    for op in pipeline.ops.values():
      tasks.extend(op_to_steps_handler(op))

    return tasks

  def _create_pipeline_workflow(self, args, pipeline, op_transformers=None, pipeline_conf=None) \
          -> List[Dict[Text, Any]]:  # Tekton change, signature/return type
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

    # generate Tekton tasks from pipeline ops
    tasks = self._create_dag_templates(pipeline, op_transformers, params)

    # generate task reference list for Tekton pipeline
    task_refs = [
      {
        'name': t['metadata']['name'],
        'taskRef': {
            'name': t['metadata']['name']
        },
        'params': [{
            'name': p['name'],
            'value': p.get('default', '')
          } for p in t['spec'].get('params', [])
        ],
        'workspaces': [{
            'name': w['name'],
            'workspace': w['name']
          } for w in t['spec'].get('workspaces', [])
        ]
      }
      for t in tasks
    ]

    # add task dependencies
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if op.dependent_names:
        task['runAfter'] = op.dependent_names

    # process input parameters from upstream tasks
    pipeline_param_names = [p['name'] for p in params]
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      for tp in task.get('params', []):
        if tp['name'] in pipeline_param_names:
          tp['value'] = '$(params.%s)' % tp['name']
        else:
          for pp in op.inputs:
            if tp['name'] == pp.full_name:
              # replace '_' to '-' since tekton results doesn't support underscore
              tp['value'] = '$(tasks.%s.results.%s)' % (pp.op_name, pp.name.replace('_', '-'))
              break
    
    # add possible workspaces
    # pop out workspaces field if it's empty
    pipeline_workspaces = []
    for task in task_refs:
      workspaces = task.get('workspaces', [])
      if workspaces:
        for w in workspaces:
          if w['name'] not in pipeline_workspaces:
            pipeline_workspaces.append(w['name'])
      else:
        task.pop('workspaces')


    # add retries params
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if op.num_retries:
        task['retries'] = op.num_retries

    # add timeout params to task_refs, instead of task.
    pipeline_conf = pipeline.conf
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if op.timeout:
        task['timeout'] = '%ds' % op.timeout

    # generate the Tekton Pipeline document
    pipeline = {
      'apiVersion': tekton_api_version,
      'kind': 'Pipeline',
      'metadata': {
        'name': pipeline.name or 'Pipeline'
      },
      'spec': {
        'params': params,
        'tasks': task_refs
      }
    }

    # Add workspaces field if there are any defined workspace.
    if pipeline_workspaces:
      pipeline['spec']['workspaces'] = [{'name': w} for w in pipeline_workspaces]

    # append Task and Pipeline documents
    workflow = tasks + [pipeline]

    # Generate pipelinerun if generate-pipelinerun flag is enabled
    if self.generate_pipelinerun:
      pipelinerun = {
        'apiVersion': tekton_api_version,
        'kind': 'PipelineRun',
        'metadata': {
          'name': pipeline['metadata']['name'] + '-run'
        },
        'spec': {
          'params': [{
            'name': p['name'],
            'value': p['default']
          } for p in pipeline['spec']['params']
          ],
          'pipelineRef': {
            'name': pipeline['metadata']['name']
          }
        }
      }

      # Mount workspaces to emptyDir if exist
      if pipeline_workspaces:
        pipelinerun['spec']['workspaces'] = [{'name': w, 'emptyDir': {}} for w in pipeline_workspaces]

      # add workflow level timeout to pipeline run
      if pipeline_conf.timeout:
        pipelinerun['spec']['timeout'] = '%ds' % pipeline_conf.timeout

      workflow = workflow + [pipelinerun]

    return workflow  # Tekton change, from return type Dict[Text, Any] to List[Dict[Text, Any]]

  # NOTE: the methods below are "copied" from KFP with changes in the method signatures (only)
  #       to accommodate multiple documents in the YAML output file:
  #         KFP Argo -> Dict[Text, Any]
  #         KFP Tekton -> List[Dict[Text, Any]]

  def _create_workflow(self,
      pipeline_func: Callable,
      pipeline_name: Text=None,
      pipeline_description: Text=None,
      params_list: List[dsl.PipelineParam]=None,
      pipeline_conf: dsl.PipelineConf = None,
      ) -> List[Dict[Text, Any]]:  # Tekton change, signature
    """ Internal implementation of create_workflow."""
    params_list = params_list or []
    argspec = inspect.getfullargspec(pipeline_func)

    # Create the arg list with no default values and call pipeline function.
    # Assign type information to the PipelineParam
    pipeline_meta = _extract_pipeline_metadata(pipeline_func)
    pipeline_meta.name = pipeline_name or pipeline_meta.name
    pipeline_meta.description = pipeline_description or pipeline_meta.description
    pipeline_name = sanitize_k8s_name(pipeline_meta.name)

    # Need to first clear the default value of dsl.PipelineParams. Otherwise, it
    # will be resolved immediately in place when being to each component.
    default_param_values = {}
    for param in params_list:
      default_param_values[param.name] = param.value
      param.value = None

    # Currently only allow specifying pipeline params at one place.
    if params_list and pipeline_meta.inputs:
      raise ValueError('Either specify pipeline params in the pipeline function, or in "params_list", but not both.')

    args_list = []
    for arg_name in argspec.args:
      arg_type = None
      for input in pipeline_meta.inputs or []:
        if arg_name == input.name:
          arg_type = input.type
          break
      args_list.append(dsl.PipelineParam(sanitize_k8s_name(arg_name, True), param_type=arg_type))

    with dsl.Pipeline(pipeline_name) as dsl_pipeline:
      pipeline_func(*args_list)

    pipeline_conf = pipeline_conf or dsl_pipeline.conf # Configuration passed to the compiler is overriding. Unfortunately, it's not trivial to detect whether the dsl_pipeline.conf was ever modified.

    self._validate_exit_handler(dsl_pipeline)
    self._sanitize_and_inject_artifact(dsl_pipeline, pipeline_conf)

    # Fill in the default values.
    args_list_with_defaults = []
    if pipeline_meta.inputs:
      args_list_with_defaults = [dsl.PipelineParam(sanitize_k8s_name(arg_name, True))
                                 for arg_name in argspec.args]
      if argspec.defaults:
        for arg, default in zip(reversed(args_list_with_defaults), reversed(argspec.defaults)):
          arg.value = default.value if isinstance(default, dsl.PipelineParam) else default
    elif params_list:
      # Or, if args are provided by params_list, fill in pipeline_meta.
      for param in params_list:
        param.value = default_param_values[param.name]

      args_list_with_defaults = params_list
      pipeline_meta.inputs = [
        InputSpec(
            name=param.name,
            type=param.param_type,
            default=param.value) for param in params_list]

    op_transformers = [add_pod_env]
    op_transformers.extend(pipeline_conf.op_transformers)

    workflow = self._create_pipeline_workflow(
        args_list_with_defaults,
        dsl_pipeline,
        op_transformers,
        pipeline_conf,
    )

    from ._data_passing_rewriter import fix_big_data_passing
    workflow = fix_big_data_passing(workflow)

    import json
    pipeline = [item for item in workflow if item["kind"] == "Pipeline"][0]  # Tekton change
    pipeline.setdefault('metadata', {}).setdefault('annotations', {})['pipelines.kubeflow.org/pipeline_spec'] = json.dumps(pipeline_meta.to_dict(), sort_keys=True)

    return workflow

  def compile(self,
              pipeline_func,
              package_path,
              type_check=True,
              pipeline_conf: dsl.PipelineConf = None,
              generate_pipelinerun=False):
    """Compile the given pipeline function into workflow yaml.
    Args:
      pipeline_func: pipeline functions with @dsl.pipeline decorator.
      package_path: the output workflow tar.gz file path. for example, "~/a.tar.gz"
      type_check: whether to enable the type check or not, default: False.
      pipeline_conf: PipelineConf instance. Can specify op transforms,
                     image pull secrets and other pipeline-level configuration options.
                     Overrides any configuration that may be set by the pipeline.
      generate_pipelinerun: Generate pipelinerun yaml for Tekton pipeline compilation.
    """
    self.generate_pipelinerun = generate_pipelinerun
    super().compile(pipeline_func, package_path, type_check, pipeline_conf=pipeline_conf)

  @staticmethod
  def _write_workflow(workflow: List[Dict[Text, Any]],  # Tekton change, signature
                      package_path: Text = None):
    """Dump pipeline workflow into yaml spec and write out in the format specified by the user.

    Args:
      workflow: Workflow spec of the pipline, dict.
      package_path: file path to be written. If not specified, a yaml_text string
        will be returned.
    """
    yaml.Dumper.ignore_aliases = lambda *args : True
    yaml_text = yaml.dump_all(workflow, default_flow_style=False)  # Tekton change

    if '{{pipelineparam' in yaml_text:
      raise RuntimeError(
          'Internal compiler error: Found unresolved PipelineParam. '
          'Please create a new issue at https://github.com/kubeflow/pipelines/issues '
          'attaching the pipeline code and the pipeline package.' )

    if package_path is None:
      return yaml_text

    if package_path.endswith('.tar.gz') or package_path.endswith('.tgz'):
      from contextlib import closing
      from io import BytesIO
      with tarfile.open(package_path, "w:gz") as tar:
          with closing(BytesIO(yaml_text.encode())) as yaml_file:
            tarinfo = tarfile.TarInfo('pipeline.yaml')
            tarinfo.size = len(yaml_file.getvalue())
            tar.addfile(tarinfo, fileobj=yaml_file)
    elif package_path.endswith('.zip'):
      with zipfile.ZipFile(package_path, "w") as zip:
        zipinfo = zipfile.ZipInfo('pipeline.yaml')
        zipinfo.compress_type = zipfile.ZIP_DEFLATED
        zip.writestr(zipinfo, yaml_text)
    elif package_path.endswith('.yaml') or package_path.endswith('.yml'):
      with open(package_path, 'w') as yaml_file:
        yaml_file.write(yaml_text)
    else:
      raise ValueError(
          'The output path '+ package_path +
          ' should ends with one of the following formats: '
          '[.tar.gz, .tgz, .zip, .yaml, .yml]')

  def _create_and_write_workflow(
      self,
      pipeline_func: Callable,
      pipeline_name: Text=None,
      pipeline_description: Text=None,
      params_list: List[dsl.PipelineParam]=None,
      pipeline_conf: dsl.PipelineConf=None,
      package_path: Text=None
  ) -> None:
    """Compile the given pipeline function and dump it to specified file format."""
    workflow = self._create_workflow(
        pipeline_func,
        pipeline_name,
        pipeline_description,
        params_list,
        pipeline_conf)
    TektonCompiler._write_workflow(workflow=workflow, package_path=package_path)   # Tekton change
