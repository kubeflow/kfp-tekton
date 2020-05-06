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
import copy
import itertools
import zipfile
import re
import logging
import textwrap
from typing import Callable, Set, List, Text, Dict, Tuple, Any, Union, Optional

from ._op_to_template import _op_to_template

from kfp import dsl
from kfp.compiler._default_transformers import add_pod_env
from kfp.compiler._k8s_helper import sanitize_k8s_name
from kfp.compiler.compiler import Compiler
# from kfp.components._yaml_utils import dump_yaml
from kfp.components.structures import InputSpec
from kfp.dsl._metadata import _extract_pipeline_metadata
from kfp.compiler._k8s_helper import convert_k8s_obj_to_json

from .. import tekton_api_version


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

  def __init__(self, **kwargs):
    # Intentionally set self.generate_pipeline and self.enable_artifacts to None because when _create_pipeline_workflow is called directly 
    # (e.g. in the case of there being no pipeline decorator), self.generate_pipeline and self.enable_artifacts is not set
    self.generate_pipelinerun = None
    self.enable_artifacts = None
    super().__init__(**kwargs)

  def _get_loop_task(self, task: Dict, op_name_to_for_loop_op):
    """Get the list of task references which will flatten the loop parameters defined in pipeline.

    Args:
      task: ops template in pipeline.
      op_name_to_for_loop_op: a dictionary of ospgroup
    """
    # Get all the params in the task
    task_parms_list = []
    for tp in task.get('params', []):
      task_parms_list.append(tp)
    # Get the loop values for each params
    for tp in task_parms_list:
      for loop_param in op_name_to_for_loop_op.values():
        loop_args = loop_param.loop_args
        if loop_args.name in tp['name']:
          lpn = tp['name'].replace(loop_args.name, '').replace('-subvar-', '')
          if lpn:
            tp['loopvalue'] = [value[lpn] for value in loop_args.items_or_pipeline_param]
          else:
            tp['loopvalue'] = loop_args.items_or_pipeline_param
    # Get the task params list
    ## Get the task_params list without loop first
    loop_value = [p['loopvalue'] for p in task_parms_list if p.get('loopvalue')]
    task_params_without_loop = [p for p in task_parms_list if not p.get('loopvalue')]
    ## Get the task_params list with loop
    loop_params = [p for p in task_parms_list if p.get('loopvalue')]
    for parm in loop_params:
      del parm['loopvalue']
      del parm['value']
    value_iter = list(itertools.product(*loop_value))
    value_iter_list = []
    for values in value_iter:
      opt = []
      for value in values:
        opt.append({"value": str(value)})
      value_iter_list.append(opt)
    {value[i].update(loop_params[i]) for i in range(len(loop_params)) for value in value_iter_list}
    task_params_with_loop = value_iter_list
    ## combine task params
    list(a.extend(task_params_without_loop) for a in task_params_with_loop)
    task_parms_all = task_params_with_loop
    # Get the task list based on parmas list
    task_list = []
    del task['params']
    task_old_name = task['name']
    for i in range(len(task_parms_all)):
      task['params'] = task_parms_all[i]
      task['name'] = '%s-loop-items-%d' % (task_old_name, i)
      task_list.append(copy.deepcopy(task))
      del task['params']
    return task_list

  def _resolve_value_or_reference(self, value_or_reference, potential_references):
    """_resolve_value_or_reference resolves values and PipelineParams, which could be task parameters or input parameters.
    Args:
      value_or_reference: value or reference to be resolved. It could be basic python types or PipelineParam
      potential_references(dict{str->str}): a dictionary of parameter names to task names
      """
    if isinstance(value_or_reference, dsl.PipelineParam):
      parameter_name = self._pipelineparam_full_name(value_or_reference)
      task_names = [task_name for param_name, task_name in potential_references if param_name == parameter_name]
      if task_names:
        task_name = task_names[0]
        # When the task_name is None, the parameter comes directly from ancient ancesters
        # instead of parents. Thus, it is resolved as the input parameter in the current group.
        if task_name is None:
          return '$(params.%s)' % parameter_name
        else:
          logging.warning("Warning: Using parameter passing from task outputs to condition parameters requires running the pipeline on Tekton built from the master branch")
          return '$(params.%s)' % task_name
      else:
        return '$(params.%s)' % parameter_name
    else:
      return str(value_or_reference)

  def _group_to_dag_template(self, group, inputs, outputs, dependencies):
    """Generate template given an OpsGroup.
    inputs, outputs, dependencies are all helper dicts.
    """

    # Generate GroupOp template
    sub_group = group
    template = {
      'apiVersion': tekton_api_version,
      'metadata': {
        'name': sub_group.name,
      },
      'spec': {}
    }
      
    # Generates template sections unique to conditions
    if isinstance(sub_group, dsl.OpsGroup) and sub_group.type == 'condition':
      subgroup_inputs = inputs.get(sub_group.name, [])
      condition = sub_group.condition

      operand1_value = self._resolve_value_or_reference(condition.operand1, subgroup_inputs)
      operand2_value = self._resolve_value_or_reference(condition.operand2, subgroup_inputs)

      # Adds params to condition template
      params = []
      if isinstance(condition.operand1, dsl.PipelineParam):
        params.append({'name': operand1_value[9: len(operand1_value) - 1]}) # Substring to remove parameter reference wrapping
      if isinstance(condition.operand2, dsl.PipelineParam):
        params.append({'name': operand2_value[9: len(operand1_value) - 1]}) # Substring to remove parameter reference wrapping
      if params:
        template['spec']['params'] = params

      # Surround the operands by quotes to ensure proper parsing on script execution
      operand1_value = '\'' + operand1_value + '\''
      operand2_value = '\'' + operand2_value + '\''

      input_grab = 'EXITCODE=$(python -c \'import sys\ninput1=str.rstrip(sys.argv[1])\ninput2=str.rstrip(sys.argv[2])\n'
      try_catch = 'try:\n  input1=int(input1)\n  input2=int(input2)\nexcept:\n  input1=str(input1)\n'
      if_else = 'print(0) if (input1 ' + condition.operator + ' input2) else print(1)\' ' + operand1_value + ' ' + operand2_value + '); '
      exit_code = 'exit $EXITCODE'
      shell_script = input_grab + try_catch + if_else + exit_code
      
      template['apiVersion'] = 'tekton.dev/v1alpha1'   # TODO Change to tekton_api_version once Conditions are out of v1alpha1
      template['kind'] = 'Condition'
      template['spec']['check'] = {
        'args': [shell_script],
        'command': ['sh','-c'],
        'image': 'python:alpine3.6',
      }

    return template


  def _create_dag_templates(self, pipeline, op_transformers=None, params=None, op_to_templates_handler=None):
    """Create all groups and ops templates in the pipeline.

    Args:
      pipeline: Pipeline context object to get all the pipeline data from.
      op_transformers: A list of functions that are applied to all ContainerOp instances that are being processed.
      op_to_templates_handler: Handler which converts a base op into a list of argo templates.
    """

    op_to_steps_handler = op_to_templates_handler or (lambda op: [_op_to_template(op, self.enable_artifacts)])
    root_group = pipeline.groups[0]

    # Call the transformation functions before determining the inputs/outputs, otherwise
    # the user would not be able to use pipeline parameters in the container definition
    # (for example as pod labels) - the generated template is invalid.
    for op in pipeline.ops.values():
      for transformer in op_transformers or []:
        transformer(op)

    # Generate core data structures to prepare for argo yaml generation
    #   op_name_to_parent_groups: op name -> list of ancestor groups including the current op
    #   opsgroups: a dictionary of ospgroup.name -> opsgroup
    #   inputs, outputs: group/op names -> list of tuples (full_param_name, producing_op_name)
    #   condition_params: recursive_group/op names -> list of pipelineparam
    #   dependencies: group/op name -> list of dependent groups/ops.
    # Special Handling for the recursive opsgroup
    #   op_name_to_parent_groups also contains the recursive opsgroups
    #   condition_params from _get_condition_params_for_ops also contains the recursive opsgroups
    #   groups does not include the recursive opsgroups
    opsgroups = self._get_groups(root_group)
    op_name_to_parent_groups = self._get_groups_for_ops(root_group)
    opgroup_name_to_parent_groups = self._get_groups_for_opsgroups(root_group)
    condition_params = self._get_condition_params_for_ops(root_group)
    inputs, outputs = self._get_inputs_outputs(
      pipeline,
      root_group,
      op_name_to_parent_groups,
      opgroup_name_to_parent_groups,
      condition_params,
      {}
    )
    dependencies = self._get_dependencies(
      pipeline,
      root_group,
      op_name_to_parent_groups,
      opgroup_name_to_parent_groups,
      opsgroups,
      condition_params,
    )

    templates = []
    for opsgroup in opsgroups.keys():
      # Only Conditions get templates in Tekton
      if opsgroups[opsgroup].type == 'condition':
        template = self._group_to_dag_template(opsgroups[opsgroup], inputs, outputs, dependencies)
        templates.append(template)

    for op in pipeline.ops.values():
      templates.extend(op_to_steps_handler(op))

    return templates


  def _process_resourceOp(self, task_refs, pipeline):
    """ handle resourceOp cases in pipeline """
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if isinstance(op, dsl.ResourceOp):
        action = op.resource.get('action')
        merge_strategy = op.resource.get('merge_strategy')
        success_condition = op.resource.get('successCondition')
        failure_condition = op.resource.get('failureCondition')
        task['params'] = [tp for tp in task.get('params', []) if tp.get('name') != "image"]
        if not merge_strategy:
          task['params'] = [tp for tp in task.get('params', []) if tp.get('name') != 'merge-strategy']
        if not success_condition:
          task['params'] = [tp for tp in task.get('params', []) if tp.get('name') != 'success-condition']
        if not failure_condition:
          task['params'] = [tp for tp in task.get('params', []) if tp.get('name') != "failure-condition"]
        for tp in task.get('params', []):
          if tp.get('name') == "action" and action:
            tp['value'] = action
          if tp.get('name') == "merge-strategy" and merge_strategy:
            tp['value'] = merge_strategy
          if tp.get('name') == "success-condition" and success_condition:
            tp['value'] = success_condition
          if tp.get('name') == "failure-condition" and failure_condition:
            tp['value'] = failure_condition
          if tp.get('name') == "output":
            output_values = ''
            for value in sorted(list(op.attribute_outputs.items()), key=lambda x: x[0]):
              output_value = textwrap.dedent("""\
                    - name: %s
                      valueFrom: '%s'
              """ % (value[0], value[1]))
              output_values += output_value
            tp['value'] = output_values


  def _workflow_with_pipelinerun(self, task_refs, pipeline, pipeline_template, workflow):
    """ Generate pipelinerun template """
    pipelinerun = {
      'apiVersion': tekton_api_version,
      'kind': 'PipelineRun',
      'metadata': {
        'name': pipeline_template['metadata']['name'] + '-run'
      },
      'spec': {
        'params': [{
          'name': p['name'],
          'value': p.get('default', '')
        } for p in pipeline_template['spec']['params']
        ],
        'pipelineRef': {
          'name': pipeline_template['metadata']['name']
        }
      }
    }

    # Generate PodTemplate
    pod_template = {}
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if op.affinity:
        pod_template['affinity'] = convert_k8s_obj_to_json(op.affinity)
      if op.tolerations:
        pod_template['tolerations'] = pod_template.get('tolerations', []) + op.tolerations
      if op.node_selector:
        pod_template['nodeSelector'] = op.node_selector

    if pod_template:
      pipelinerun['spec']['podtemplate'] = pod_template

    # add workflow level timeout to pipeline run
    if pipeline.conf.timeout:
      pipelinerun['spec']['timeout'] = '%ds' % pipeline.conf.timeout

    # generate the Tekton service account template for image pull secret
    service_template = {}
    if len(pipeline.conf.image_pull_secrets) > 0:
      service_template = {
        'apiVersion': 'v1',
        'kind': 'ServiceAccount',
        'metadata': {'name': pipelinerun['metadata']['name'] + '-sa'}
      }
    for image_pull_secret in pipeline.conf.image_pull_secrets:
      service_template['imagePullSecrets'] = [{'name': image_pull_secret.name}]

    if service_template:
      workflow = workflow + [service_template]
      pipelinerun['spec']['serviceAccountName'] = service_template['metadata']['name']

    workflow = workflow + [pipelinerun]

    return workflow


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
    templates = self._create_dag_templates(pipeline, op_transformers, params)

    # generate task and condition reference list for the Tekton Pipeline
    condition_refs = {}
    task_refs = []
    for template in templates:
      if template['kind'] == 'Condition':
        condition_refs[template['metadata']['name']] = [{
          'conditionRef': template['metadata']['name'],
          'params': [{
              'name': param['name'],
              'value': '$(params.'+param['name']+')'
            } for param in template['spec'].get('params',[])
          ]
        }]
      else:
        task_refs.append(
          {
            'name': template['metadata']['name'],
            'taskRef': {
              'name': template['metadata']['name']
            },
            'params': [{
                'name': p['name'],
                'value': p.get('default', '')
              } for p in template['spec'].get('params', [])
            ]
          }
        )
    
    # process input parameters from upstream tasks for conditions and pair conditions with their ancestor conditions
    opsgroup_stack = [pipeline.groups[0]]
    condition_stack = [None]
    while opsgroup_stack:
      cur_opsgroup = opsgroup_stack.pop()
      most_recent_condition = condition_stack.pop()

      if cur_opsgroup.type == 'condition':
        condition_ref = condition_refs[cur_opsgroup.name][0]
        condition = cur_opsgroup.condition
        input_params = []

        # Process input parameters if needed
        if isinstance(condition.operand1, dsl.PipelineParam):
          if condition.operand1.op_name:
            operand_value = '$(tasks.'+condition.operand1.op_name+'.results.'+condition.operand1.name+')'
          else:
            operand_value = '$(params.'+condition.operand1.name+')'
          input_params.append(operand_value)
        if isinstance(condition.operand2, dsl.PipelineParam):
          if condition.operand2.op_name:
            operand_value = '$(tasks.'+condition.operand2.op_name+'.results.'+condition.operand2.name+')'
          else:
            operand_value = '$(params.'+condition.operand2.name+')'
          input_params.append(operand_value)
        for param_iter in range(len(input_params)):
          condition_ref['params'][param_iter]['value'] = input_params[param_iter]

        # Add ancestor conditions to the current condition ref
        if most_recent_condition:
          condition_refs[cur_opsgroup.name].extend(condition_refs[most_recent_condition])
        most_recent_condition = cur_opsgroup.name

      opsgroup_stack.extend(cur_opsgroup.groups)
      condition_stack.extend([most_recent_condition for x in range(len(cur_opsgroup.groups))])


    # add task dependencies and add condition refs to the task ref that depends on the condition
    op_name_to_parent_groups = self._get_groups_for_ops(pipeline.groups[0])
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      parent_group = op_name_to_parent_groups.get(task['name'], [])
      if parent_group:
        if condition_refs.get(parent_group[-2],[]):
          task['conditions'] = condition_refs.get(op_name_to_parent_groups[task['name']][-2],[])
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

    # add retries params
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if op.num_retries:
        task['retries'] = op.num_retries

    # add timeout params to task_refs, instead of task.
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if op.timeout:
        task['timeout'] = '%ds' % op.timeout

    # handle resourceOp cases in pipeline
    self._process_resourceOp(task_refs, pipeline)

    # process loop parameters, keep this section in the behind of other processes, ahead of gen pipeline
    root_group = pipeline.groups[0]
    op_name_to_for_loop_op = self._get_for_loop_ops(root_group)
    if op_name_to_for_loop_op:
      for loop_param in op_name_to_for_loop_op.values():
        if loop_param.items_is_pipeline_param is True:
          raise NotImplementedError("dynamic params are not yet implemented")
      include_loop_task_refs = []
      for task in task_refs:
        with_loop_task = self._get_loop_task(task, op_name_to_for_loop_op)
        include_loop_task_refs.extend(with_loop_task)
      task_refs = include_loop_task_refs

    # generate the Tekton Pipeline document
    pipeline_template = {
      'apiVersion': tekton_api_version,
      'kind': 'Pipeline',
      'metadata': {
        'name': pipeline.name or 'Pipeline',
        'annotations': {
          # Disable Istio inject since Tekton cannot run with Istio sidecar
          'sidecar.istio.io/inject': 'false'
        }
      },
      'spec': {
        'params': params,
        'tasks': task_refs
      }
    }

    # append Task and Pipeline documents
    workflow = templates + [pipeline_template]

    # Generate pipelinerun if generate-pipelinerun flag is enabled
    if self.generate_pipelinerun:
      workflow = self._workflow_with_pipelinerun(task_refs, pipeline, pipeline_template, workflow)

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
              generate_pipelinerun=False,
              enable_artifacts=False):
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
    self.enable_artifacts = enable_artifacts
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
    # yaml_text = dump_yaml(workflow)
    yaml.Dumper.ignore_aliases = lambda *args : True
    yaml_text = yaml.dump_all(workflow, default_flow_style=False)  # Tekton change

    # Use regex to replace all the Argo variables to Tekton variables. For variables that are unique to Argo,
    # we raise an Error to alert users about the unsupported variables. Here is the list of Argo variables.
    # https://github.com/argoproj/argo/blob/master/docs/variables.md
    # Since Argo variables can be used in anywhere in the yaml, we need to dump and then parse the whole yaml
    # using regular expression.
    tekton_var_regex_rules = [
        {'argo_rule': '{{inputs.parameters.([^ \t\n.:,;{}]+)}}', 'tekton_rule': '$(inputs.params.\g<1>)'},
        {'argo_rule': '{{outputs.parameters.([^ \t\n.:,;{}]+).path}}', 'tekton_rule': '$(results.\g<1>.path)'}
    ]
    for regex_rule in tekton_var_regex_rules:
      yaml_text = re.sub(regex_rule['argo_rule'], regex_rule['tekton_rule'], yaml_text)

    unsupported_vars = re.findall(r"{{[^ \t\n.:,;{}]+\.[^ \t\n:,;{}]+}}", yaml_text)
    if unsupported_vars:
      raise ValueError('These Argo variables are not supported in Tekton Pipeline: %s' % ", ".join(str(v) for v in set(unsupported_vars)))

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
    _validate_workflow(workflow)


def _validate_workflow(workflow: List[Dict[Text, Any]]):  # Tekton change, signature
  # TODO: Tekton pipeline parameter validation
#   workflow = workflow.copy()
#   # Working around Argo lint issue
#   for argument in workflow['spec'].get('arguments', {}).get('parameters', []):
#     if 'value' not in argument:
#       argument['value'] = ''
#
#   yaml_text = dump_yaml(workflow)
#   if '{{pipelineparam' in yaml_text:
#     raise RuntimeError(
#         '''Internal compiler error: Found unresolved PipelineParam.
# Please create a new issue at https://github.com/kubeflow/pipelines/issues attaching the pipeline code and the pipeline package.'''
#     )

  # TODO: Tekton lint, if a tool exists for it
#   # Running Argo lint if available
#   import shutil
#   import subprocess
#   argo_path = shutil.which('argo')
#   if argo_path:
#     result = subprocess.run([argo_path, 'lint', '/dev/stdin'], input=yaml_text.encode('utf-8'), stdout=subprocess.PIPE, stderr=subprocess.PIPE)
#     if result.returncode:
#       raise RuntimeError(
#         '''Internal compiler error: Compiler has produced Argo-incompatible workflow.
# Please create a new issue at https://github.com/kubeflow/pipelines/issues attaching the pipeline code and the pipeline package.
# Error: {}'''.format(result.stderr.decode('utf-8'))
#       )
  pass
