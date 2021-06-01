# Copyright 2019-2021 kubeflow.org.
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
import zipfile
import re
import textwrap
import yaml
import os
import uuid
import ast

from typing import Callable, List, Text, Dict, Any
from os import environ as env
from distutils.util import strtobool

# Kubeflow Pipeline imports
from kfp import dsl
from kfp.compiler._default_transformers import add_pod_env
from kfp.compiler.compiler import Compiler
from kfp.components.structures import InputSpec
from kfp.dsl._for_loop import LoopArguments
from kfp.dsl._metadata import _extract_pipeline_metadata
from collections import defaultdict

# KFP-Tekton imports
from kfp_tekton.compiler import __tekton_api_version__ as tekton_api_version
from kfp_tekton.compiler._data_passing_rewriter import fix_big_data_passing
from kfp_tekton.compiler._k8s_helper import convert_k8s_obj_to_json, sanitize_k8s_name, sanitize_k8s_object
from kfp_tekton.compiler._op_to_template import _op_to_template
from kfp_tekton.compiler.yaml_utils import dump_yaml
from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf
from kfp_tekton.compiler._tekton_handler import _handle_tekton_pipeline_variables, _handle_tekton_custom_task
from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES, DEFAULT_CONDITION_OUTPUT_KEYWORD

DEFAULT_ARTIFACT_BUCKET = env.get('DEFAULT_ARTIFACT_BUCKET', 'mlpipeline')
DEFAULT_ARTIFACT_ENDPOINT = env.get('DEFAULT_ARTIFACT_ENDPOINT', 'minio-service.kubeflow:9000')
DEFAULT_ARTIFACT_ENDPOINT_SCHEME = env.get('DEFAULT_ARTIFACT_ENDPOINT_SCHEME', 'http://')
TEKTON_GLOBAL_DEFAULT_TIMEOUT = strtobool(env.get('TEKTON_GLOBAL_DEFAULT_TIMEOUT', 'false'))
# DISABLE_CEL_CONDITION should be True until CEL is officially merged into Tekton main API.
DISABLE_CEL_CONDITION = True


def _get_super_condition_template():

  python_script = textwrap.dedent('''\
    'import sys
    input1=str.rstrip(sys.argv[1])
    input2=str.rstrip(sys.argv[2])
    try:
      input1=int(input1)
      input2=int(input2)
    except:
      input1=str(input1)
    %(s)s="true" if (input1 $(inputs.params.operator) input2) else "false"
    f = open("/tekton/results/%(s)s", "w")
    f.write(%(s)s)
    f.close()' '''
    % {'s': DEFAULT_CONDITION_OUTPUT_KEYWORD})

  template = {
    'results': [
      {'name': DEFAULT_CONDITION_OUTPUT_KEYWORD,
       'description': 'Conditional task %s' % DEFAULT_CONDITION_OUTPUT_KEYWORD
       }
    ],
    'params': [
      {'name': 'operand1'},
      {'name': 'operand2'},
      {'name': 'operator'}
    ],
    'steps': [{
      'script': 'python -c ' + python_script + "'$(inputs.params.operand1)' '$(inputs.params.operand2)'",
      'image': 'python:alpine3.6',
    }]
  }

  return template


def _get_cel_condition_template():
  template = {
    "name": "cel_condition",
    "apiVersion": "cel.tekton.dev/v1alpha1",
    "kind": "CEL"
  }

  return template


class TektonCompiler(Compiler):
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
    # Input and output artifacts are hash maps for metadata tracking.
    # artifact_items is the artifact dependency map
    # loops_pipeline recorde the loop tasks information for each loops
    self.input_artifacts = {}
    self.output_artifacts = {}
    self.artifact_items = {}
    self.loops_pipeline = {}
    self.recursive_tasks = []
    self.custom_task_crs = []
    self.uuid = self._get_unique_id_code()
    self._group_names = []
    self.pipeline_labels = {}
    self.pipeline_annotations = {}
    super().__init__(**kwargs)

  def _set_pipeline_conf(self, tekton_pipeline_conf: TektonPipelineConf):
    self.pipeline_labels = tekton_pipeline_conf.pipeline_labels
    self.pipeline_annotations = tekton_pipeline_conf.pipeline_annotations

  def _resolve_value_or_reference(self, value_or_reference, potential_references):
    """_resolve_value_or_reference resolves values and PipelineParams, which could be task parameters or input parameters.
    Args:
      value_or_reference: value or reference to be resolved. It could be basic python types or PipelineParam
      potential_references(dict{str->str}): a dictionary of parameter names to task names
      """
    if isinstance(value_or_reference, dsl.PipelineParam):
      parameter_name = value_or_reference.full_name
      task_names = [task_name for param_name, task_name in potential_references if param_name == parameter_name]
      if task_names:
        task_name = task_names[0]
        # When the task_name is None, the parameter comes directly from ancient ancesters
        # instead of parents. Thus, it is resolved as the input parameter in the current group.
        if task_name is None:
          return '$(params.%s)' % parameter_name
        else:
          return '$(params.%s)' % task_name
      else:
        return '$(params.%s)' % parameter_name
    else:
      return str(value_or_reference)

  def _get_groups(self, root_group):
    """Helper function to get all groups (not including ops) in a pipeline."""

    def _get_groups_helper(group):
      groups = {group.name: group}
      for g in group.groups:
        groups.update(_get_groups_helper(g))
      return groups

    return _get_groups_helper(root_group)

  @staticmethod
  def _get_unique_id_code():
    return uuid.uuid4().hex[:5]

  def _group_to_dag_template(self, group, inputs, outputs, dependencies, pipeline_name, group_type):
    """Generate template given an OpsGroup.
    inputs, outputs, dependencies are all helper dicts.
    """
    # Generate GroupOp template
    sub_group = group
    # For loop and recursion usually append 16-19 characters, so limit the loop/recusion pipeline_name to 44 char
    self._group_names = [sanitize_k8s_name(pipeline_name, max_length=44), sanitize_k8s_name(sub_group.name)]
    if self.uuid:
      self._group_names.insert(1, self.uuid)
    group_name = '-'.join(self._group_names) if group_type == "loop" or group_type == "graph" else sub_group.name
    template = {
      'metadata': {
        'name': group_name,
      },
      'spec': {}
    }

    # Generates a pseudo-template unique to conditions due to the catalog condition approach
    # where every condition is an extension of one super-condition
    if isinstance(sub_group, dsl.OpsGroup) and sub_group.type == 'condition':
      subgroup_inputs = inputs.get(group_name, [])
      condition = sub_group.condition

      operand1_value = self._resolve_value_or_reference(condition.operand1, subgroup_inputs)
      operand2_value = self._resolve_value_or_reference(condition.operand2, subgroup_inputs)
      template['kind'] = 'Condition'
      template['spec']['params'] = [
        {'name': 'operand1', 'value': operand1_value, 'type': type(condition.operand1),
        'op_name': getattr(condition.operand1, 'op_name', ''), 'output_name': getattr(condition.operand1, 'name', '')},
        {'name': 'operand2', 'value': operand2_value, 'type': type(condition.operand2),
        'op_name': getattr(condition.operand2, 'op_name', ''), 'output_name': getattr(condition.operand2, 'name', '')},
        {'name': 'operator', 'value': str(condition.operator), 'type': type(condition.operator)}
      ]

    # dsl does not expose Graph so here use sub_group.type to check whether it's graph
    if sub_group.type == "graph":
      # for graph now we just support as a pipeline loop with just 1 iteration
      loop_args_name = "just_one_iteration"
      loop_args_value = ["1"]

      # Special handling for recursive subgroup
      if sub_group.recursive_ref:
        # generate ref graph name
        tmp_group_names = [pipeline_name, sanitize_k8s_name(sub_group.recursive_ref.name)]
        if self.uuid:
          tmp_group_names.insert(1, self.uuid)
        ref_group_name = '-'.join(tmp_group_names)

        # generate params
        params = [{
          "name": loop_args_name,
          "value": loop_args_value
        }]

        # get other input params, for recursion need rename the param name to the refrenced one
        for i in range(len(sub_group.inputs)):
            input = sub_group.inputs[i]
            inputRef = sub_group.recursive_ref.inputs[i]
            if input.op_name:
              params.append({
                'name': inputRef.full_name,
                'value': '$(tasks.%s.results.%s)' % (input.op_name, input.name)
              })
            else:
              params.append({
                'name': inputRef.full_name, 'value': '$(params.%s)' % input.name
              })

        self.recursive_tasks.append({
          'name': sub_group.name,
          'taskRef': {
            'apiVersion': 'custom.tekton.dev/v1alpha1',
            'kind': 'PipelineLoop',
            'name': ref_group_name
          },
          'params': params
        })
      # normal graph logic start from here
      else:
        self.loops_pipeline[group_name] = {
          'kind': 'loops',
          'loop_args': loop_args_name,
          'loop_sub_args': [],
          'task_list': [],
          'spec': {},
          'depends': []
        }
        # get the dependencies tasks rely on the loop task.
        for depend in dependencies.keys():
          if depend == sub_group.name:
            self.loops_pipeline[group_name]['spec']['runAfter'] = [task for task in dependencies[depend]]
            self.loops_pipeline[group_name]['spec']['runAfter'].sort()
          # for items depend on the graph, it will be handled in custom task handler
          if sub_group.name in dependencies[depend]:
            dependencies[depend].remove(sub_group.name)
            self.loops_pipeline[group_name]['depends'].append({'org': depend, 'runAfter': group_name})
        for op in sub_group.groups + sub_group.ops:
          self.loops_pipeline[group_name]['task_list'].append(sanitize_k8s_name(op.name))
          if hasattr(op, 'type') and op.type == 'condition':
            if op.ops:
              for condition_op in op.ops:
                self.loops_pipeline[group_name]['task_list'].append(sanitize_k8s_name(condition_op.name))
            if op.groups:
              for condition_op in op.groups:
                # graph task should be a nested cr and need be appended to task list.
                if condition_op.type == 'graph':
                  self.loops_pipeline[group_name]['task_list'].append(sanitize_k8s_name(condition_op.name))
        self.loops_pipeline[group_name]['spec']['name'] = group_name
        self.loops_pipeline[group_name]['spec']['taskRef'] = {
          "apiVersion": "custom.tekton.dev/v1alpha1",
          "kind": "PipelineLoop",
          "name": group_name
        }

        self.loops_pipeline[group_name]['spec']['params'] = [{
          "name": loop_args_name,
          "value": loop_args_value
        }]

        # get other input params
        for input in inputs.keys():
          if input == sub_group.name:
            for param in inputs[input]:
              if param[1]:
                replace_str = param[1] + '-'
                self.loops_pipeline[group_name]['spec']['params'].append({
                  'name': param[0], 'value': '$(tasks.%s.results.%s)' % (
                    param[1], sanitize_k8s_name(param[0].replace(replace_str, ''))
                  )
                })
              if not param[1]:
                self.loops_pipeline[group_name]['spec']['params'].append({
                  'name': param[0], 'value': '$(params.%s)' % param[0]
                })

    if isinstance(sub_group, dsl.ParallelFor):
      self.loops_pipeline[group_name] = {
        'kind': 'loops',
        'loop_args': sub_group.loop_args.full_name,
        'loop_sub_args': [],
        'task_list': [],
        'spec': {},
        'depends': []
      }
      for subvarName in sub_group.loop_args.referenced_subvar_names:
        if subvarName != '__iter__':
          self.loops_pipeline[group_name]['loop_sub_args'].append(sub_group.loop_args.full_name + '-subvar-' + subvarName)
      if isinstance(sub_group.loop_args.items_or_pipeline_param, list) and isinstance(
        sub_group.loop_args.items_or_pipeline_param[0], dict):
        for key in sub_group.loop_args.items_or_pipeline_param[0]:
          self.loops_pipeline[group_name]['loop_sub_args'].append(sub_group.loop_args.full_name + '-subvar-' + key)
      # get the dependencies tasks rely on the loop task.
      for depend in dependencies.keys():
        if depend == sub_group.name:
          self.loops_pipeline[group_name]['spec']['runAfter'] = [task for task in dependencies[depend]]
          self.loops_pipeline[group_name]['spec']['runAfter'].sort()
        if sub_group.name in dependencies[depend]:
          self.loops_pipeline[group_name]['depends'].append({'org': depend, 'runAfter': group_name})
      for op in sub_group.groups + sub_group.ops:
        self.loops_pipeline[group_name]['task_list'].append(sanitize_k8s_name(op.name))
        if hasattr(op, 'type') and op.type == 'condition' and op.ops:
          for condition_op in op.ops:
            self.loops_pipeline[group_name]['task_list'].append(sanitize_k8s_name(condition_op.name))
      self.loops_pipeline[group_name]['spec']['name'] = group_name
      self.loops_pipeline[group_name]['spec']['taskRef'] = {
        "apiVersion": "custom.tekton.dev/v1alpha1",
        "kind": "PipelineLoop",
        "name": group_name
      }
      if sub_group.items_is_pipeline_param:
        # these loop args are a 'dynamic param' rather than 'static param'.
        # i.e., rather than a static list, they are either the output of another task or were input
        # as global pipeline parameters
        pipeline_param = sub_group.loop_args.items_or_pipeline_param
        if pipeline_param.op_name is None:
          withparam_value = '$(params.%s)' % pipeline_param.name
        else:
          param_name = sanitize_k8s_name(pipeline_param.name)
          withparam_value = '$(tasks.%s.results.%s)' % (
            sanitize_k8s_name(pipeline_param.op_name),
            param_name)
        self.loops_pipeline[group_name]['spec']['params'] = [{
          "name": sub_group.loop_args.full_name,
          "value": withparam_value
        }]
      else:
        # Need to sanitize the dict keys for consistency.
        loop_arg_value = sub_group.loop_args.to_list_for_task_yaml()
        loop_args_str_value = ''
        sanitized_tasks = []
        if isinstance(loop_arg_value[0], dict):
          for argument_set in loop_arg_value:
            c_dict = {}
            for k, v in argument_set.items():
              c_dict[sanitize_k8s_name(k, True)] = v
            sanitized_tasks.append(c_dict)
          loop_args_str_value = json.dumps(sanitized_tasks, sort_keys=True)
        else:
          loop_args_str_value = str(loop_arg_value)

        self.loops_pipeline[group_name]['spec']['params'] = [{
          "name": sub_group.loop_args.full_name,
          "value": loop_args_str_value
        }]
      # get other input params
      for input in inputs.keys():
        if input == sub_group.name:
          for param in inputs[input]:
            if param[0] != sub_group.loop_args.full_name and param[1] and param[0] not in self.loops_pipeline[group_name][
              'loop_sub_args']:
              replace_str = param[1] + '-'
              self.loops_pipeline[group_name]['spec']['params'].append({
                'name': param[0], 'value': '$(tasks.%s.results.%s)' % (
                  param[1], sanitize_k8s_name(param[0].replace(replace_str, ''))
                )
              })
            if param[0] != sub_group.loop_args.full_name and not param[1] and param[0] not in self.loops_pipeline[group_name][
              'loop_sub_args']:
              self.loops_pipeline[group_name]['spec']['params'].append({
                'name': param[0], 'value': '$(params.%s)' % param[0]
              })
      if sub_group.parallelism is not None:
        self.loops_pipeline[group_name]['spec']['parallelism'] = sub_group.parallelism

    return template

  def _create_dag_templates(self, pipeline, op_transformers=None, params=None, op_to_templates_handler=None):
    """Create all groups and ops templates in the pipeline.

    Args:
      pipeline: Pipeline context object to get all the pipeline data from.
      op_transformers: A list of functions that are applied to all ContainerOp instances that are being processed.
      op_to_templates_handler: Handler which converts a base op into a list of argo templates.
    """

    op_to_steps_handler = op_to_templates_handler or (lambda op: [_op_to_template(op,
                                                                  self.output_artifacts,
                                                                  self.artifact_items)])
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
    op_name_to_for_loop_op = self._get_for_loop_ops(root_group)
    inputs, outputs = self._get_inputs_outputs(
      pipeline,
      root_group,
      op_name_to_parent_groups,
      opgroup_name_to_parent_groups,
      condition_params,
      op_name_to_for_loop_op
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
      # Conditions and loops will get templates in Tekton
      if opsgroups[opsgroup].type == 'condition':
        template = self._group_to_dag_template(opsgroups[opsgroup], inputs, outputs, dependencies, pipeline.name, "condition")
        templates.append(template)
      if opsgroups[opsgroup].type == 'for_loop':
        self._group_to_dag_template(opsgroups[opsgroup], inputs, outputs, dependencies, pipeline.name, "loop")
      if opsgroups[opsgroup].type == 'graph':
        self._group_to_dag_template(opsgroups[opsgroup], inputs, outputs, dependencies, pipeline.name, "graph")

    for op in pipeline.ops.values():
      templates.extend(op_to_steps_handler(op))

    return templates

  def _get_inputs_outputs(
          self,
          pipeline,
          root_group,
          op_groups,
          opsgroup_groups,
          condition_params,
          op_name_to_for_loop_op: Dict[Text, dsl.ParallelFor],
  ):
    """Get inputs and outputs of each group and op.
    Returns:
      A tuple (inputs, outputs).
      inputs and outputs are dicts with key being the group/op names and values being list of
      tuples (param_name, producing_op_name). producing_op_name is the name of the op that
      produces the param. If the param is a pipeline param (no producer op), then
      producing_op_name is None.
    """
    inputs = defaultdict(set)
    outputs = defaultdict(set)

    for op in pipeline.ops.values():
      # op's inputs and all params used in conditions for that op are both considered.
      for param in op.inputs + list(condition_params[op.name]):
        # if the value is already provided (immediate value), then no need to expose
        # it as input for its parent groups.
        if param.value:
          continue
        if param.op_name:
          upstream_op = pipeline.ops[param.op_name]
          upstream_groups, downstream_groups = \
            self._get_uncommon_ancestors(op_groups, opsgroup_groups, upstream_op, op)
          for i, group_name in enumerate(downstream_groups):
            # Important: Changes for Tekton custom tasks
            # Custom task condition are not pods running in Tekton. Thus it should also
            # be considered as the first uncommon downstream group.
            def is_parent_custom_task(index):
              for group_name in downstream_groups[:index]:
                if 'condition-' in group_name:
                  return True
              return False
            if i == 0 or is_parent_custom_task(i):
              # If it is the first uncommon downstream group, then the input comes from
              # the first uncommon upstream group.
              inputs[group_name].add((param.full_name, upstream_groups[0]))
            else:
              # If not the first downstream group, then the input is passed down from
              # its ancestor groups so the upstream group is None.
              inputs[group_name].add((param.full_name, None))
          for i, group_name in enumerate(upstream_groups):
            if i == len(upstream_groups) - 1:
              # If last upstream group, it is an operator and output comes from container.
              outputs[group_name].add((param.full_name, None))
            else:
              # If not last upstream group, output value comes from one of its child.
              outputs[group_name].add((param.full_name, upstream_groups[i + 1]))
        else:
          if not op.is_exit_handler:
            for group_name in op_groups[op.name][::-1]:
              # if group is for loop group and param is that loop's param, then the param
              # is created by that for loop ops_group and it shouldn't be an input to
              # any of its parent groups.
              inputs[group_name].add((param.full_name, None))
              if group_name in op_name_to_for_loop_op:
                # for example:
                #   loop_group.loop_args.name = 'loop-item-param-99ca152e'
                #   param.name =                'loop-item-param-99ca152e--a'
                loop_group = op_name_to_for_loop_op[group_name]
                if loop_group.loop_args.name in param.name:
                  break

    # Generate the input/output for recursive opsgroups
    # It propagates the recursive opsgroups IO to their ancester opsgroups
    def _get_inputs_outputs_recursive_opsgroup(group):
      # TODO: refactor the following codes with the above
      if group.recursive_ref:
        params = [(param, False) for param in group.inputs]
        params.extend([(param, True) for param in list(condition_params[group.name])])
        for param, is_condition_param in params:
          if param.value:
            continue
          full_name = param.full_name
          if param.op_name:
            upstream_op = pipeline.ops[param.op_name]
            upstream_groups, downstream_groups = \
              self._get_uncommon_ancestors(op_groups, opsgroup_groups, upstream_op, group)
            for i, g in enumerate(downstream_groups):
              if i == 0:
                inputs[g].add((full_name, upstream_groups[0]))
              # There is no need to pass the condition param as argument to the downstream ops.
              # TODO: this might also apply to ops. add a TODO here and think about it.
              elif i == len(downstream_groups) - 1 and is_condition_param:
                continue
              else:
                # For Tekton, do not append duplicated input parameters
                duplicated_downstream_group = False
                for group_name in inputs[g]:
                  if len(group_name) > 1 and group_name[0] == full_name:
                    duplicated_downstream_group = True
                    break
                if not duplicated_downstream_group:
                  inputs[g].add((full_name, None))
            for i, g in enumerate(upstream_groups):
              if i == len(upstream_groups) - 1:
                outputs[g].add((full_name, None))
              else:
                outputs[g].add((full_name, upstream_groups[i + 1]))
          elif not is_condition_param:
            for g in op_groups[group.name]:
              inputs[g].add((full_name, None))
      for subgroup in group.groups:
        _get_inputs_outputs_recursive_opsgroup(subgroup)

    _get_inputs_outputs_recursive_opsgroup(root_group)

    # Generate the input for SubGraph along with parallelfor
    for sub_graph in opsgroup_groups:
      if sub_graph in op_name_to_for_loop_op:
        # The opsgroup list is sorted with the farthest group as the first and
        # the opsgroup itself as the last. To get the latest opsgroup which is
        # not the opsgroup itself -2 is used.
        parent = opsgroup_groups[sub_graph][-2]
        if parent and parent.startswith('subgraph'):
          # propagate only op's pipeline param from subgraph to parallelfor
          loop_op = op_name_to_for_loop_op[sub_graph]
          pipeline_param = loop_op.loop_args.items_or_pipeline_param
          if loop_op.items_is_pipeline_param and pipeline_param.op_name:
            param_name = '%s-%s' % (
              sanitize_k8s_name(pipeline_param.op_name), pipeline_param.name)
            inputs[parent].add((param_name, pipeline_param.op_name))

    return inputs, outputs

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

  def _create_pipeline_workflow(self, args, pipeline, op_transformers=None, pipeline_conf=None) \
          -> Dict[Text, Any]:
    """Create workflow for the pipeline."""
    # Input Parameters
    params = []
    for arg in args:
      param = {'name': arg.name}
      if arg.value is not None:
        if isinstance(arg.value, (list, tuple, dict)):
          param['default'] = json.dumps(arg.value, sort_keys=True)
        else:
          param['default'] = str(arg.value)
      params.append(param)

    # generate Tekton tasks from pipeline ops
    raw_templates = self._create_dag_templates(pipeline, op_transformers, params)

    # generate task and condition reference list for the Tekton Pipeline
    condition_refs = {}

    task_refs = []
    templates = []
    cel_conditions = {}
    condition_when_refs = {}
    condition_task_refs = {}
    string_condition_refs = {}
    for template in raw_templates:
      if template['kind'] == 'Condition':
        if DISABLE_CEL_CONDITION:
          condition_task_spec = _get_super_condition_template()
        else:
          condition_task_spec = _get_cel_condition_template()

        condition_params = template['spec'].get('params', [])
        if condition_params:
          condition_task_ref = [{
              'name': template['metadata']['name'],
              'params': [{
                  'name': p['name'],
                  'value': p.get('value', '')
                } for p in template['spec'].get('params', [])
              ],

              'taskSpec' if DISABLE_CEL_CONDITION else 'taskRef': condition_task_spec
          }]
          condition_refs[template['metadata']['name']] = [
              {
                'input': '$(tasks.%s.results.%s)' % (template['metadata']['name'], DEFAULT_CONDITION_OUTPUT_KEYWORD),
                'operator': 'in',
                'values': ['true']
              }
            ]
          # Don't use additional task if it's only doing literal string == and !=
          # with CEL custom task output.
          condition_operator = condition_params[2]
          condition_operand1 = condition_params[0]
          condition_operand2 = condition_params[1]
          conditionOp_mapping = {"==": "in", "!=": "notin"}
          if condition_operator.get('value', '') in conditionOp_mapping.keys():
            # Check whether the operand is an output from custom task
            # If so, don't create a new task to verify the condition.
            def is_custom_task_output(operand) -> bool:
              if operand['type'] == dsl.PipelineParam:
                for template in raw_templates:
                  if operand['op_name'] == template['metadata']['name']:
                    for step in template['spec']['steps']:
                      if step['name'] == 'main' and step['image'] in TEKTON_CUSTOM_TASK_IMAGES:
                        return True
              return False
            if is_custom_task_output(condition_operand1) or is_custom_task_output(condition_operand2):
              map_cel_vars = lambda a: '$(tasks.%s.results.%s)' % (sanitize_k8s_name(a['value'].split('.')[-1]),
                sanitize_k8s_name(a['output_name'])) if a.get('type', '') == dsl.PipelineParam else a.get('value', '')
              condition_refs[template['metadata']['name']] = [
                  {
                    'input': map_cel_vars(condition_operand1),
                    'operator': conditionOp_mapping[condition_operator['value']],
                    'values': [map_cel_vars(condition_operand2)]
                  }
                ]
              string_condition_refs[template['metadata']['name']] = True
          condition_task_refs[template['metadata']['name']] = condition_task_ref
          condition_when_refs[template['metadata']['name']] = condition_refs[template['metadata']['name']]
      else:
        templates.append(template)
        task_ref = {
            'name': template['metadata']['name'],
            'params': [{
                'name': p['name'],
                'value': p.get('default', '')
              } for p in template['spec'].get('params', [])
            ],
            'taskSpec': template['spec'],
          }

        for i in template['spec'].get('steps', []):
          # TODO: change the below conditions to map with a label
          #       or a list of images with optimized actions
          if i.get('image', '') in TEKTON_CUSTOM_TASK_IMAGES:
            custom_task_args = {}
            container_args = i.get('args', [])
            for index, item in enumerate(container_args):
              if item.startswith('--'):
                custom_task_args[item[2:]] = container_args[index + 1]
            non_param_keys = ['name', 'apiVersion', 'kind', 'taskSpec']
            task_params = []
            for key, value in custom_task_args.items():
              if key not in non_param_keys:
                task_params.append({'name': key, 'value': value})
            task_ref = {
              'name': template['metadata']['name'],
              'params': task_params,
              # For processing Tekton parameter mapping later on.
              'orig_params': task_ref['params'],
              'taskRef': {
                'name': custom_task_args['name'],
                'apiVersion': custom_task_args['apiVersion'],
                'kind': custom_task_args['kind']
              }
            }
            if custom_task_args.get('taskSpec', ''):
              try:
                if custom_task_args['taskSpec']:
                  custom_task_cr = {
                    'apiVersion': custom_task_args['apiVersion'],
                    'kind': custom_task_args['kind'],
                    'metadata': {
                      'name': custom_task_args['name']
                    },
                    'spec': ast.literal_eval(custom_task_args['taskSpec'])
                  }
                  for existing_cr in self.custom_task_crs:
                    if existing_cr == custom_task_cr:
                      # Skip duplicated CR resource
                      custom_task_cr = {}
                      break
                  if custom_task_cr:
                    self.custom_task_crs.append(custom_task_cr)
              except ValueError:
                raise("Custom task spec %s is not a valid Python Dictionary" % custom_task_args['taskSpec'])
            # Pop custom task artifacts since we have no control of how
            # custom task controller is handling the container/task execution.
            self.artifact_items.pop(template['metadata']['name'], None)
            self.output_artifacts.pop(template['metadata']['name'], None)
            break
        if task_ref.get('taskSpec', ''):
          task_ref['taskSpec']['metadata'] = task_ref['taskSpec'].get('metadata', {})
          task_labels = template['metadata'].get('labels', {})
          task_ref['taskSpec']['metadata']['labels'] = task_labels
          task_labels['pipelines.kubeflow.org/pipelinename'] = task_labels.get('pipelines.kubeflow.org/pipelinename', '')
          task_labels['pipelines.kubeflow.org/generation'] = task_labels.get('pipelines.kubeflow.org/generation', '')
          cache_default = self.pipeline_labels.get('pipelines.kubeflow.org/cache_enabled', 'true')
          task_labels['pipelines.kubeflow.org/cache_enabled'] = task_labels.get('pipelines.kubeflow.org/cache_enabled', cache_default)

          task_annotations = template['metadata'].get('annotations', {})
          task_ref['taskSpec']['metadata']['annotations'] = task_annotations
          task_annotations['tekton.dev/template'] = task_annotations.get('tekton.dev/template', '')

        task_refs.append(task_ref)

    # process input parameters from upstream tasks for conditions and pair conditions with their ancestor conditions
    opsgroup_stack = [pipeline.groups[0]]
    condition_stack = [None]
    while opsgroup_stack:
      cur_opsgroup = opsgroup_stack.pop()
      most_recent_condition = condition_stack.pop()

      if cur_opsgroup.type == 'condition':
        condition_task_ref = condition_task_refs[cur_opsgroup.name][0]
        condition = cur_opsgroup.condition
        input_params = []
        if not cel_conditions.get(condition_task_ref['name'], None):
          # Process input parameters if needed
          if isinstance(condition.operand1, dsl.PipelineParam):
            if condition.operand1.op_name:
              operand_value = '$(tasks.' + condition.operand1.op_name + '.results.' + sanitize_k8s_name(condition.operand1.name) + ')'
            else:
              operand_value = '$(params.' + condition.operand1.name + ')'
            input_params.append(operand_value)
          if isinstance(condition.operand2, dsl.PipelineParam):
            if condition.operand2.op_name:
              operand_value = '$(tasks.' + condition.operand2.op_name + '.results.' + sanitize_k8s_name(condition.operand2.name) + ')'
            else:
              operand_value = '$(params.' + condition.operand2.name + ')'
            input_params.append(operand_value)
        for param_iter in range(len(input_params)):
          # Add ancestor conditions to the current condition ref
          if most_recent_condition:
            add_ancestor_conditions = True
            # Do not add ancestor conditions if the ancestor is not in the same graph/pipelineloop
            for pipeline_loop in self.loops_pipeline.values():
              if condition_task_ref['name'] in pipeline_loop['task_list']:
                if most_recent_condition not in pipeline_loop['task_list']:
                  add_ancestor_conditions = False
            if add_ancestor_conditions:
              condition_task_ref['when'] = condition_when_refs[most_recent_condition]
          condition_task_ref['params'][param_iter]['value'] = input_params[param_iter]
        if not DISABLE_CEL_CONDITION and not cel_conditions.get(condition_task_ref['name'], None):
          # Type processing are done on the CEL controller since v1 SDK doesn't have value type for conditions.
          # For v2 SDK, it would be better to process the condition value type in the backend compiler.
          var1 = condition_task_ref['params'][0]['value']
          var2 = condition_task_ref['params'][1]['value']
          op = condition_task_ref['params'][2]['value']
          condition_task_ref['params'] = [{
                  'name': DEFAULT_CONDITION_OUTPUT_KEYWORD,
                  'value': " ".join([var1, op, var2])
                }]
        most_recent_condition = cur_opsgroup.name
      opsgroup_stack.extend(cur_opsgroup.groups)
      condition_stack.extend([most_recent_condition for x in range(len(cur_opsgroup.groups))])
    # add task dependencies and add condition refs to the task ref that depends on the condition
    op_name_to_parent_groups = self._get_groups_for_ops(pipeline.groups[0])
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      parent_group = op_name_to_parent_groups.get(task['name'], [])
      if parent_group:
        if condition_refs.get(parent_group[-2], []):
          task['when'] = condition_refs.get(op_name_to_parent_groups[task['name']][-2], [])
      if op != None and op.dependent_names:
        task['runAfter'] = op.dependent_names

    # add condition refs to the recursive refs that depends on the condition
    for recursive_task in self.recursive_tasks:
      parent_group = op_name_to_parent_groups.get(recursive_task['name'], [])
      if parent_group:
        if condition_refs.get(parent_group[-2], []):
          recursive_task['when'] = condition_refs.get(op_name_to_parent_groups[recursive_task['name']][-2], [])
      recursive_task['name'] = sanitize_k8s_name(recursive_task['name'])

    # add condition refs to the pipelineloop refs that depends on the condition
    opgroup_name_to_parent_groups = self._get_groups_for_opsgroups(pipeline.groups[0])
    for loop_task_key in self.loops_pipeline.keys():
      task_name_prefix = '-'.join(self._group_names[:-1] + [""])
      raw_task_key = loop_task_key.replace(task_name_prefix, "")
      parent_group = opgroup_name_to_parent_groups.get(raw_task_key, [])
      if parent_group:
        if condition_refs.get(parent_group[-2], []):
          self.loops_pipeline[loop_task_key]['spec']['when'] = condition_refs.get(parent_group[-2], [])
          for i, param in enumerate(self.loops_pipeline[loop_task_key]['spec']["params"]):
            if param["value"] == condition_refs.get(parent_group[-2], [])[0]["input"]:
              self.loops_pipeline[loop_task_key]['spec']["params"].pop(i)
              break

    # process input parameters from upstream tasks
    pipeline_param_names = [p['name'] for p in params]
    loop_args = [self.loops_pipeline[key]['loop_args'] for key in self.loops_pipeline.keys()]
    for key in self.loops_pipeline.keys():
      if self.loops_pipeline[key]['loop_sub_args'] != []:
        loop_args.extend(self.loops_pipeline[key]['loop_sub_args'])
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      # Substitute task paramters to the correct Tekton variables.
      # Regular task and custom task have different parameter mapping in Tekton.
      if task.get('orig_params', []):  # custom task
        orig_params = [p['name'] for p in task.get('orig_params', [])]
        for tp in task.get('params', []):
          pipeline_params = re.findall('\$\(inputs.params.([^ \t\n.:,;{}]+)\)', tp.get('value', ''))
          # There could be multiple pipeline params in one expression, so we need to map each of them
          # back to the proper tekton variables.
          for pipeline_param in pipeline_params:
            if pipeline_param in orig_params:
              if pipeline_param in pipeline_param_names + loop_args:
                # Do not sanitize Tekton pipeline input parameters, only the output parameters need to be sanitized
                substitute_param = '$(params.%s)' % pipeline_param
                tp['value'] = re.sub('\$\(inputs.params.%s\)' % pipeline_param, substitute_param, tp.get('value', ''))
              else:
                for pp in op.inputs:
                  if pipeline_param == pp.full_name:
                    # Parameters from Tekton results need to be sanitized
                    substitute_param = '$(tasks.%s.results.%s)' % (sanitize_k8s_name(pp.op_name), sanitize_k8s_name(pp.name))
                    tp['value'] = re.sub('\$\(inputs.params.%s\)' % pipeline_param, substitute_param, tp.get('value', ''))
                    break
        # Not necessary for Tekton execution
        task.pop('orig_params', None)
      else:  # regular task
        op = pipeline.ops.get(task['name'])
        for tp in task.get('params', []):
          if tp['name'] in pipeline_param_names + loop_args:
            tp['value'] = '$(params.%s)' % tp['name']
          else:
            for pp in op.inputs:
              if tp['name'] == pp.full_name:
                tp['value'] = '$(tasks.%s.results.%s)' % (pp.op_name, pp.name)
                # Create input artifact tracking annotation
                input_annotation = self.input_artifacts.get(task['name'], [])
                input_annotation.append(
                    {
                        'name': tp['name'],
                        'parent_task': pp.op_name
                    }
                )
                self.input_artifacts[task['name']] = input_annotation
                break

    # add retries params
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if op != None and op.num_retries:
        task['retries'] = op.num_retries

    # add timeout params to task_refs, instead of task.
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      # Custom task doesn't support timeout feature
      if task.get('taskSpec', ''):
        if op != None and (not TEKTON_GLOBAL_DEFAULT_TIMEOUT or op.timeout):
          task['timeout'] = '%ds' % op.timeout

    # handle resourceOp cases in pipeline
    self._process_resourceOp(task_refs, pipeline)

    # handle exit handler in pipeline
    finally_tasks = []
    for task in task_refs:
      op = pipeline.ops.get(task['name'])
      if op != None and op.is_exit_handler:
        finally_tasks.append(task)
    task_refs = [task for task in task_refs if pipeline.ops.get(task['name']) and not pipeline.ops.get(task['name']).is_exit_handler]

    # Flatten condition task
    condition_task_refs_temp = []
    for condition_task_ref in condition_task_refs.values():
      for ref in condition_task_ref:
        if not string_condition_refs.get(ref['name'], False):
          condition_task_refs_temp.append(ref)
    condition_task_refs = condition_task_refs_temp

    pipeline_run = {
      'apiVersion': tekton_api_version,
      'kind': 'PipelineRun',
      'metadata': {
        'name': sanitize_k8s_name(pipeline.name or 'Pipeline', suffix_space=4),
        # Reflect the list of Tekton pipeline annotations at the top
        'annotations': {
          'tekton.dev/output_artifacts': json.dumps(self.output_artifacts, sort_keys=True),
          'tekton.dev/input_artifacts': json.dumps(self.input_artifacts, sort_keys=True),
          'tekton.dev/artifact_bucket': DEFAULT_ARTIFACT_BUCKET,
          'tekton.dev/artifact_endpoint': DEFAULT_ARTIFACT_ENDPOINT,
          'tekton.dev/artifact_endpoint_scheme': DEFAULT_ARTIFACT_ENDPOINT_SCHEME,
          'tekton.dev/artifact_items': json.dumps(self.artifact_items, sort_keys=True),
          'sidecar.istio.io/inject': 'false'  # disable Istio inject since Tekton cannot run with Istio sidecar
        }
      },
      'spec': {
        'params': [{
            'name': p['name'],
            'value': p.get('default', '')
          } for p in params],
        'pipelineSpec': {
          'params': params,
          'tasks': task_refs + condition_task_refs,
          'finally': finally_tasks
        }
      }
    }

    if self.pipeline_labels:
      pipeline_run['metadata']['labels'] = pipeline_run['metadata'].setdefault('labels', {})
      pipeline_run['metadata']['labels'].update(self.pipeline_labels)
      # Remove pipeline level label for 'pipelines.kubeflow.org/cache_enabled' as it overwrites task level label
      pipeline_run['metadata']['labels'].pop('pipelines.kubeflow.org/cache_enabled', None)

    if self.pipeline_annotations:
      pipeline_run['metadata']['annotations'] = pipeline_run['metadata'].setdefault('annotations', {})
      pipeline_run['metadata']['annotations'].update(self.pipeline_annotations)

    # Generate TaskRunSpec PodTemplate:s
    task_run_spec = []
    for task in task_refs:

      # TODO: should loop-item tasks be included here?
      if LoopArguments.LOOP_ITEM_NAME_BASE in task['name']:
        task_name = re.sub(r'-%s-.+$' % LoopArguments.LOOP_ITEM_NAME_BASE, '', task['name'])
      else:
        task_name = task['name']
      op = pipeline.ops.get(task_name)
      if not op:
        raise RuntimeError("unable to find op with name '%s'" % task["name"])

      task_spec = {"pipelineTaskName": task['name'],
                   "taskPodTemplate": {}}
      if op.affinity:
        task_spec["taskPodTemplate"]["affinity"] = convert_k8s_obj_to_json(op.affinity)
      if op.tolerations:
        task_spec["taskPodTemplate"]['tolerations'] = op.tolerations
      if op.node_selector:
        task_spec["taskPodTemplate"]['nodeSelector'] = op.node_selector
      if bool(task_spec["taskPodTemplate"]):
        task_run_spec.append(task_spec)
    if len(task_run_spec) > 0:
      pipeline_run['spec']['taskRunSpecs'] = task_run_spec

    # add workflow level timeout to pipeline run
    if not TEKTON_GLOBAL_DEFAULT_TIMEOUT or pipeline.conf.timeout:
      pipeline_run['spec']['timeout'] = '%ds' % pipeline.conf.timeout

    # generate the Tekton podTemplate for image pull secret
    if len(pipeline.conf.image_pull_secrets) > 0:
      pipeline_run['spec']['podTemplate'] = pipeline_run['spec'].get('podTemplate', {})
      pipeline_run['spec']['podTemplate']['imagePullSecrets'] = [
        {"name": s.name} for s in pipeline.conf.image_pull_secrets]

    workflow = pipeline_run

    return workflow

  def _sanitize_and_inject_artifact(self, pipeline: dsl.Pipeline, pipeline_conf=None):
    """Sanitize operator/param names and inject pipeline artifact location."""

    # Sanitize operator names and param names
    sanitized_ops = {}

    for op in pipeline.ops.values():
      if len(op.name) > 57:
        raise ValueError('Input ops cannot be longer than 57 characters. \
             \nOp name: %s' % op.name)
      sanitized_name = sanitize_k8s_name(op.name)
      op.name = sanitized_name
      # check sanitized input params
      for param in op.inputs:
        if param.op_name:
          if len(param.op_name) > 128:
            raise ValueError('Input parameter cannot be longer than 128 characters. \
             \nInput name: %s. \nOp name: %s' % (param.op_name, op.name))
          param.op_name = sanitize_k8s_name(param.op_name, max_length=float('inf'))
      # sanitized output params
      for param in op.outputs.values():
        param.name = sanitize_k8s_name(param.name, True)
        if param.op_name:
          param.op_name = sanitize_k8s_name(param.op_name)
      if op.output is not None and not isinstance(op.output, dsl._container_op._MultipleOutputsError):
        op.output.name = sanitize_k8s_name(op.output.name, True)
        op.output.op_name = sanitize_k8s_name(op.output.op_name)
      if op.dependent_names:
        op.dependent_names = [sanitize_k8s_name(name) for name in op.dependent_names]
      if isinstance(op, dsl.ContainerOp) and op.file_outputs is not None:
        sanitized_file_outputs = {}
        for key in op.file_outputs.keys():
          sanitized_file_outputs[sanitize_k8s_name(key, True)] = op.file_outputs[key]
        op.file_outputs = sanitized_file_outputs
      elif isinstance(op, dsl.ResourceOp) and op.attribute_outputs is not None:
        sanitized_attribute_outputs = {}
        for key in op.attribute_outputs.keys():
          sanitized_attribute_outputs[sanitize_k8s_name(key, True)] = \
            op.attribute_outputs[key]
        op.attribute_outputs = sanitized_attribute_outputs
      if isinstance(op, dsl.ContainerOp) and op.container is not None:
        sanitize_k8s_object(op.container)
      sanitized_ops[sanitized_name] = op
    pipeline.ops = sanitized_ops

  # NOTE: the methods below are "copied" from KFP with changes in the method signatures (only)
  #       to accommodate multiple documents in the YAML output file:
  #         KFP Argo -> Dict[Text, Any]
  #         KFP Tekton -> List[Dict[Text, Any]]

  def _create_workflow(self,
                       pipeline_func: Callable,
                       pipeline_name: Text = None,
                       pipeline_description: Text = None,
                       params_list: List[dsl.PipelineParam] = None,
                       pipeline_conf: dsl.PipelineConf = None,
                       ) -> Dict[Text, Any]:
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

    # Configuration passed to the compiler is overriding. Unfortunately, it is
    # not trivial to detect whether the dsl_pipeline.conf was ever modified.
    pipeline_conf = pipeline_conf or dsl_pipeline.conf

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

    workflow = fix_big_data_passing(workflow)

    workflow.setdefault('metadata', {}).setdefault('annotations', {})['pipelines.kubeflow.org/pipeline_spec'] = \
      json.dumps(pipeline_meta.to_dict(), sort_keys=True)

    # recursively strip empty structures, DANGER: this may remove necessary empty elements ?!
    def remove_empty_elements(obj) -> dict:
      if not isinstance(obj, (dict, list)):
        return obj
      if isinstance(obj, list):
        return [remove_empty_elements(o) for o in obj if o != []]
      return {k: remove_empty_elements(v) for k, v in obj.items()
              if v != []}

    workflow = remove_empty_elements(workflow)

    return workflow

  def compile(self,
              pipeline_func,
              package_path,
              type_check=True,
              pipeline_conf: dsl.PipelineConf = None,
              tekton_pipeline_conf: TektonPipelineConf = None):
    """Compile the given pipeline function into workflow yaml.
    Args:
      pipeline_func: pipeline functions with @dsl.pipeline decorator.
      package_path: the output workflow tar.gz file path. for example, "~/a.tar.gz"
      type_check: whether to enable the type check or not, default: False.
      pipeline_conf: PipelineConf instance. Can specify op transforms,
                     image pull secrets and other pipeline-level configuration options.
                     Overrides any configuration that may be set by the pipeline.
    """
    if tekton_pipeline_conf:
      self._set_pipeline_conf(tekton_pipeline_conf)
    super().compile(pipeline_func, package_path, type_check, pipeline_conf=pipeline_conf)

  @staticmethod
  def _write_workflow(workflow: Dict[Text, Any],
                      package_path: Text = None):
    """Dump pipeline workflow into yaml spec and write out in the format specified by the user.

    Args:
      workflow: Workflow spec of the pipline, dict.
      package_path: file path to be written. If not specified, a yaml_text string
        will be returned.
    """
    yaml_text = dump_yaml(workflow)

    # Use regex to replace all the Argo variables to Tekton variables. For variables that are unique to Argo,
    # we raise an Error to alert users about the unsupported variables. Here is the list of Argo variables.
    # https://github.com/argoproj/argo/blob/master/docs/variables.md
    # Since Argo variables can be used in anywhere in the yaml, we need to dump and then parse the whole yaml
    # using regular expression.
    tekton_var_regex_rules = [
        {
          'argo_rule': '{{inputs.parameters.([^ \t\n.:,;{}]+)}}',
          'tekton_rule': '$(inputs.params.\g<1>)'
        },
        {
          'argo_rule': '{{outputs.parameters.([^ \t\n.:,;{}]+).path}}',
          'tekton_rule': '$(results.\g<1>.path)'
        },
        {
          'argo_rule': '{{workflow.uid}}',
          'tekton_rule': '$(context.pipelineRun.uid)'
        },
        {
          'argo_rule': '{{workflow.name}}',
          'tekton_rule': '$(context.pipelineRun.name)'
        },
        {
          'argo_rule': '{{workflow.namespace}}',
          'tekton_rule': '$(context.pipelineRun.namespace)'
        },
        {
          'argo_rule': '{{workflow.parameters.([^ \t\n.:,;{}]+)}}',
          'tekton_rule': '$(params.\g<1>)'
        }
    ]
    for regex_rule in tekton_var_regex_rules:
      yaml_text = re.sub(regex_rule['argo_rule'], regex_rule['tekton_rule'], yaml_text)

    unsupported_vars = re.findall(r"{{[^ \t\n.:,;{}]+\.[^ \t\n:,;{}]+}}", yaml_text)
    if unsupported_vars:
      raise ValueError('These Argo variables are not supported in Tekton Pipeline: %s' % ", ".join(str(v) for v in set(unsupported_vars)))
    if '{{pipelineparam' in yaml_text:
      raise RuntimeError(
          'Internal compiler error: Found unresolved PipelineParam. '
          'Please create a new issue at https://github.com/kubeflow/kfp-tekton/issues '
          'attaching the pipeline DSL code and the pipeline YAML.')

    pipeline_run = yaml.load(yaml_text, Loader=yaml.FullLoader)
    if pipeline_run.get("spec", {}) and pipeline_run["spec"].get("pipelineSpec", {}) and \
      pipeline_run["spec"]["pipelineSpec"].get("tasks", []):
      yaml_text = dump_yaml(_handle_tekton_pipeline_variables(pipeline_run))

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
          'The output path %s should end with one of the following formats: '
          '[.tar.gz, .tgz, .zip, .yaml, .yml]' % package_path)

  def _create_and_write_workflow(self,
                                 pipeline_func: Callable,
                                 pipeline_name: Text = None,
                                 pipeline_description: Text = None,
                                 params_list: List[dsl.PipelineParam] = None,
                                 pipeline_conf: dsl.PipelineConf = None,
                                 package_path: Text = None,
                                 ) -> None:
    """Compile the given pipeline function and dump it to specified file format."""
    workflow = self._create_workflow(
        pipeline_func,
        pipeline_name,
        pipeline_description,
        params_list,
        pipeline_conf)
    # Separate loop workflow from the main workflow
    if self.loops_pipeline:
      pipeline_loop_crs, workflow = _handle_tekton_custom_task(self.loops_pipeline, workflow, self.recursive_tasks, self._group_names)
      TektonCompiler._write_workflow(workflow=workflow, package_path=package_path)
      for i in range(len(pipeline_loop_crs)):
        TektonCompiler._write_workflow(workflow=pipeline_loop_crs[i],
                                       package_path=os.path.splitext(package_path)[0] + "_pipelineloop_cr" + str(i + 1) + '.yaml')
    else:
      TektonCompiler._write_workflow(workflow=workflow, package_path=package_path)   # Tekton change
    # Separate custom task CR from the main workflow
    for i in range(len(self.custom_task_crs)):
      TektonCompiler._write_workflow(workflow=self.custom_task_crs[i],
                                     package_path=os.path.splitext(package_path)[0] + "_customtask_cr" + str(i + 1) + '.yaml')
    _validate_workflow(workflow)


def _validate_workflow(workflow: Dict[Text, Any]):

  # verify that all names and labels conform to kubernetes naming standards
  #   https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
  #   https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/

  def _find_items(obj, search_key, current_path="", results_dict=dict()) -> dict:
    if isinstance(obj, dict):
      if search_key in obj:
        results_dict.update({"%s.%s" % (current_path, search_key): obj[search_key]})
      for k, v in obj.items():
        _find_items(v, search_key, "%s.%s" % (current_path, k), results_dict)
    elif isinstance(obj, list):
      for i, list_item in enumerate(obj):
        _find_items(list_item, search_key, "%s[%i]" % (current_path, i), results_dict)
    return {k.lstrip("."): v for k, v in results_dict.items()}

  non_k8s_names = {path: name for path, name in _find_items(workflow, "name").items()
                   if "metadata" in path and name != sanitize_k8s_name(name, max_length=253)
                   or "param" in path and name != sanitize_k8s_name(name, allow_capital_underscore=True, max_length=253)}

  non_k8s_labels = {path: k_v_dict for path, k_v_dict in _find_items(workflow, "labels", "", {}).items()
                    if "metadata" in path and
                    any([k != sanitize_k8s_name(k, allow_capital_underscore=True, allow_dot=True, allow_slash=True, max_length=253) or
                         v != sanitize_k8s_name(v, allow_capital_underscore=True, allow_dot=True)
                         for k, v in k_v_dict.items()])}

  non_k8s_annotations = {path: k_v_dict for path, k_v_dict in _find_items(workflow, "annotations", "", {}).items()
                         if "metadata" in path and
                         any([k != sanitize_k8s_name(k, allow_capital_underscore=True, allow_dot=True, allow_slash=True, max_length=253)
                              for k in k_v_dict.keys()])}

  error_msg_tmplt = textwrap.dedent("""\
    Internal compiler error: Found non-compliant Kubernetes %s:
    %s
    Please create a new issue at https://github.com/kubeflow/kfp-tekton/issues
    attaching the pipeline DSL code and the pipeline YAML.""")

  if non_k8s_names:
    raise RuntimeError(error_msg_tmplt % ("names", json.dumps(non_k8s_names, sort_keys=False, indent=2)))

  if non_k8s_labels:
    raise RuntimeError(error_msg_tmplt % ("labels", json.dumps(non_k8s_labels, sort_keys=False, indent=2)))

  if non_k8s_annotations:
    raise RuntimeError(error_msg_tmplt % ("annotations", json.dumps(non_k8s_annotations, sort_keys=False, indent=2)))

  # TODO: Tekton pipeline parameter validation
  #   workflow = workflow.copy()
  #   # Working around Argo lint issue
  #   for argument in workflow['spec'].get('arguments', {}).get('parameters', []):
  #     if 'value' not in argument:
  #       argument['value'] = ''
  #   yaml_text = dump_yaml(workflow)
  #   if '{{pipelineparam' in yaml_text:
  #     raise RuntimeError(
  #         '''Internal compiler error: Found unresolved PipelineParam.
  # Please create a new issue at https://github.com/kubeflow/kfp-tekton/issues
  # attaching the pipeline code and the pipeline package.'''
  #     )

  # TODO: Tekton lint, if a tool exists for it
  #   # Running Argo lint if available
  #   import shutil
  #   import subprocess
  #   argo_path = shutil.which('argo')
  #   if argo_path:
  #     result = subprocess.run([argo_path, 'lint', '/dev/stdin'], input=yaml_text.encode('utf-8'),
  #                             stdout=subprocess.PIPE, stderr=subprocess.PIPE)
  #     if result.returncode:
  #       raise RuntimeError(
  #         '''Internal compiler error: Compiler has produced Argo-incompatible workflow.
  # Please create a new issue at https://github.com/kubeflow/kfp-tekton/issues
  # attaching the pipeline code and the pipeline package.
  # Error: {}'''.format(result.stderr.decode('utf-8'))
  #       )
  pass
