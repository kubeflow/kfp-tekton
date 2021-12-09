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

import json, copy, re
from kfp_tekton.compiler._k8s_helper import sanitize_k8s_name

from kfp_tekton.tekton import LOOP_GROUP_NAME_LENGTH


def _process_argo_vars(workflow):
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
        workflow = re.sub(regex_rule['argo_rule'], regex_rule['tekton_rule'], workflow)
    return workflow


def _handle_tekton_pipeline_variables(pipeline_run):
    """
    Handle tekton pipeline level variables, such as context.pipelineRun.name
    See more here https://github.com/tektoncd/pipeline/blob/master/docs/variables.md
    If there are tekton pipeline level variables in container, then
    1. Replease the $(context.pipeline/pipelineRun).*) to $(params.var_name)
       for example context.pipelineRun.name -> $(params.pipelineRun-name)
    2. Add $(context.pipeline/pipelineRun).*) to pipeline_run['spec']['pipelineSpec']['tasks']['params']
    3. Add $(context.pipeline/pipelineRun).*) to pipeline_run['spec']['pipelineSpec']['tasks']['taskSpec']['params']
    """

    pipeline_variables = {
        'pipeline-name': '$(context.pipeline.name)',
        'pipelineRun-name': '$(context.pipelineRun.name)',
        'pipelineRun-namespace': '$(context.pipelineRun.namespace)',
        'pipelineRun-uid': '$(context.pipelineRun.uid)'
    }

    task_list = pipeline_run['spec']['pipelineSpec']['tasks']
    for task in task_list:
        if task.get('taskRef', {}):
            continue
        if 'taskSpec' in task and 'apiVersion' in task['taskSpec']:
            continue
        for key, val in pipeline_variables.items():
            task_str = json.dumps(task['taskSpec']['steps'])
            task_str = _process_argo_vars(task_str)
            if val in task_str:
                task_str = task_str.replace(val, '$(params.' + key + ')')
                task['taskSpec']['steps'] = json.loads(task_str)
                if task.get('params', ''):
                    if {'name': key, 'value': val} not in task['params']:
                        task['params'].append({'name': key, 'value': val})
                else:
                    task['params'] = [{'name': key, 'value': val}]
                if task['taskSpec'].get('params', ''):
                    if {'name': key} not in task['taskSpec']['params']:
                        task['taskSpec']['params'].append({'name': key})
                else:
                    task['taskSpec']['params'] = [{'name': key}]

    return pipeline_run


def _handle_tekton_custom_task(custom_task: dict, workflow: dict, recursive_tasks: list, group_names: list):
    """
    Separate custom task's workflow from the main workflow, return a tuple result of custom task cr definitions
    and a new workflow

    Args:
      custom_task: dictionary with custom_task infomation, the format should be as below:
      {
        'kind': '',
        'task_list': [],
        'spec': {},
        'depends': []
      }
      workflow: a workflow without loop pipeline separeted.
      recursive_tasks: List of recursive_tasks information.
      group_names: List of name constructions for creating custom loop crd names.

    Returns:
      A tuple (custom_task_crs, workflow).
      custom_task_crs is a list of custom task cr definitions.
      and workflow is a dict which will not including the tasks in custom task definitions
    """
    custom_task_crs = []
    task_list = []
    tasks = workflow['spec']['pipelineSpec']['tasks']
    new_tasks = []
    dependencies = []
    # handle dependecies
    for key in custom_task.keys():
        dependencies.extend(custom_task[key]['depends'])
    for task in tasks:
        for dependency in dependencies:
            if task['name'] == dependency['org']:
                task_dependencies = [dependency['runAfter']]
                for dep_task in task.get('runAfter', []):
                    # should exclude the custom task itself for cases like graph
                    dep_task_trim = copy.copy(dep_task)
                    if len(group_names[-1]) <= LOOP_GROUP_NAME_LENGTH:
                        dep_task_trim = sanitize_k8s_name(dep_task, max_length=LOOP_GROUP_NAME_LENGTH, rev_truncate=True)
                    dep_task_with_prefix = '-'.join(group_names[:-1] + [dep_task_trim])
                    if dep_task_with_prefix == dependency['runAfter']:
                        continue
                    if dep_task not in custom_task[dependency['runAfter']]['task_list']:
                        task_dependencies.append(dep_task)
                task['runAfter'] = task_dependencies

    # process recursive tasks to match parameters
    for task in recursive_tasks:
        recursive_graph = custom_task.get(task['taskRef']['name'], {})
        if recursive_graph:
            if recursive_graph['spec']['params']:
                recursive_graph['spec']['params'] = sorted(recursive_graph['spec']['params'], key=lambda k: k['name'])
            for param in recursive_graph['spec']['params']:
                recursive_params = [param['name'] for param in task['params']]
                if param['name'] not in recursive_params:
                    task['params'].append({'name': param['name'], 'value': "$(params.%s)" % param['name']})

    # get custom tasks
    for custom_task_key in custom_task.keys():
        denpendency_list = custom_task[custom_task_key]['spec'].get('runAfter', [])
        task_list.extend(custom_task[custom_task_key]['task_list'])
        # generate custom task cr
        custom_task_cr_tasks = []
        for task in tasks:
            if task['name'] in custom_task[custom_task_key]['task_list']:
                for param in task.get('taskSpec', {}).get('params', []):
                    param['type'] = 'string'
                run_after_task_list = []
                for run_after_task in task.get('runAfter', []):
                    for recursive_task in recursive_tasks:
                        # The subset of the loop group name should be LOOP_GROUP_NAME_LENGTH minus 4 because the
                        # numbers of loop cannot exceed 1000 due to ETCD limitation.
                        if sanitize_k8s_name(recursive_task['name'], max_length=(LOOP_GROUP_NAME_LENGTH - 4), rev_truncate=True) \
                            in run_after_task and '-'.join(group_names[:-1]) not in run_after_task:
                            if len(group_names[-1]) <= LOOP_GROUP_NAME_LENGTH:
                                run_after_task = sanitize_k8s_name(run_after_task, max_length=LOOP_GROUP_NAME_LENGTH, rev_truncate=True)
                            run_after_task = '-'.join(group_names[:-1] + [run_after_task])
                            break
                    if run_after_task not in denpendency_list:
                        # check task name is task list
                        for origin_task_name in task_list:
                            if origin_task_name in run_after_task:
                                run_after_task_list.append(run_after_task)
                                break
                if task.get('runAfter', []):
                    task['runAfter'] = run_after_task_list
                custom_task_cr_tasks.append(task)
        # append recursive tasks
        for task in recursive_tasks:
            if task['name'] in custom_task[custom_task_key]['task_list']:
                custom_task_cr_tasks.append(task)
        # generator custom task cr
        custom_task_cr = {
            "apiVersion": "custom.tekton.dev/v1alpha1",
            "kind": 'custom_task_kind',
            "metadata": {
                "name": custom_task_key
            },
            "spec": {
                "pipelineSpec": {
                    "params": [{
                        "name": parm['name'],
                        'type': 'string'
                    } for parm in sorted(custom_task[custom_task_key]['spec']['params'], key=lambda k: k['name'])],
                    "tasks": custom_task_cr_tasks
                }
            }
        }

        # handle loop special case
        if custom_task[custom_task_key]['kind'] == 'loops':
            # if subvar exist, this is dict loop parameters
            # remove the loop_arg and add subvar args to the cr params
            if custom_task[custom_task_key]['loop_sub_args'] != []:
                refesh_cr_params = []
                for param in custom_task_cr['spec']['pipelineSpec']['params']:
                    if param['name'] != custom_task[custom_task_key]['loop_args']:
                        refesh_cr_params.append(param)
                custom_task_cr['spec']['pipelineSpec']['params'] = refesh_cr_params
                custom_task_cr['spec']['pipelineSpec']['params'].extend([{
                    "name": sub_param,
                    'type': 'string'
                } for sub_param in custom_task[custom_task_key]['loop_sub_args']])

            # add loop special filed
            custom_task_cr['kind'] = 'PipelineLoop'
            if custom_task[custom_task_key]['spec'].get('parallelism') is not None:
                custom_task_cr['spec']['parallelism'] = custom_task[custom_task_key]['spec']['parallelism']
                # remove from pipeline run spec
                del custom_task[custom_task_key]['spec']['parallelism']
            custom_task_cr['spec']['iterateParam'] = custom_task[custom_task_key]['loop_args']
            for custom_task_param in custom_task[custom_task_key]['spec']['params']:
                if custom_task_param['name'] != custom_task[custom_task_key]['loop_args'] and '$(tasks.' in custom_task_param['value']:
                    custom_task_cr = json.loads(
                        json.dumps(custom_task_cr).replace(custom_task_param['value'], '$(params.%s)' % custom_task_param['name']))

        # need to process task parameters to replace out of scope results
        # because nested graph cannot refer to task results outside of the sub-pipeline.
        custom_task_cr_task_names = [custom_task_cr_task['name'] for custom_task_cr_task in custom_task_cr['spec']['pipelineSpec']['tasks']]
        for task in custom_task_cr['spec']['pipelineSpec']['tasks']:
            for task_param in task.get('params', []):
                if '$(tasks.' in task_param['value']:
                    param_results = re.findall('\$\(tasks.([^ \t\n.:,;\{\}]+).results.([^ \t\n.:,;\{\}]+)\)', task_param['value'])
                    for param_result in param_results:
                        if param_result[0] not in custom_task_cr_task_names:
                            task['params'] = json.loads(
                                json.dumps(task['params']).replace(task_param['value'],
                                '$(params.%s-%s)' % param_result))
        custom_task_crs.append(custom_task_cr)
        custom_task[custom_task_key]['spec']['params'] = sorted(custom_task[custom_task_key]['spec']['params'],
                                                                          key=lambda k: k['name'])
        tasks.append(custom_task[custom_task_key]['spec'])

    # handle the nested custom task case
    # Need to be verified: nested custom task with tasks result as parameters
    nested_custom_tasks = []
    custom_task_crs_namelist = []
    for custom_task_key in custom_task.keys():
        if len(group_names[-1]) <= LOOP_GROUP_NAME_LENGTH:
            sanitize_k8s_name(custom_task_key, max_length=LOOP_GROUP_NAME_LENGTH, rev_truncate=True)
        custom_task_crs_namelist.append(custom_task_key)
    for custom_task_key in custom_task.keys():
        for inner_task_name in custom_task[custom_task_key]['task_list']:
            inner_task_name_trimmed = copy.copy(inner_task_name)
            if len(group_names[-1]) <= LOOP_GROUP_NAME_LENGTH:
                inner_task_name_trimmed = sanitize_k8s_name(inner_task_name, max_length=LOOP_GROUP_NAME_LENGTH, rev_truncate=True)
            inner_task_cr_name = '-'.join(group_names[:-1] + [inner_task_name_trimmed])
            if inner_task_cr_name in custom_task_crs_namelist:
                nested_custom_tasks.append({
                    "father_ct": custom_task_key,
                    "nested_custom_task": inner_task_cr_name
                })
    # Summary out all of the nested tasks relationships.
    for nested_custom_task in nested_custom_tasks:
        father_ct_name = nested_custom_task['father_ct']
        relationships = find_ancestors(nested_custom_tasks, father_ct_name, [], father_ct_name)
        nested_custom_task['ancestors'] = relationships['ancestors']
        nested_custom_task['root_ct'] = relationships['root_ct']

    for nested_custom_task in nested_custom_tasks:
        nested_custom_task_spec = custom_task[nested_custom_task['nested_custom_task']]['spec']
        for custom_task_cr in custom_task_crs:
            if custom_task_cr['metadata']['name'] == nested_custom_task['father_ct']:
                # handle parameters of nested custom task
                params_nested_custom_task = nested_custom_task_spec['params']
                # nested_custom_task_special_params = the global params that doesn't defined in parent custom task
                nested_custom_task_special_params = [
                    param for param in params_nested_custom_task
                    if '$(params.' in param['value'] and not bool([
                        True for ct_param in custom_task_cr['spec']['pipelineSpec']['params']
                        if param['name'] in ct_param['name']
                    ])
                ]
                custom_task_cr['spec']['pipelineSpec']['params'].extend([
                    {'name': param['name'], 'type': 'string'}for param in nested_custom_task_special_params
                ])

                if nested_custom_task['ancestors']:
                    for custom_task_cr_again in custom_task_crs:
                        if custom_task_cr_again['metadata']['name'] in nested_custom_task[
                            'ancestors'] or custom_task_cr_again['metadata']['name'] == nested_custom_task['root_ct']:
                            custom_task_cr_again['spec']['pipelineSpec']['params'].extend([
                                {'name': param['name'], 'type': 'string'}for param in nested_custom_task_special_params
                            ])
                            custom_task_cr_again['spec']['pipelineSpec']['params'] = sorted(
                                custom_task_cr_again['spec']['pipelineSpec']['params'], key=lambda k: k['name'])
                # add children params to the root tasks
                for task in tasks:
                    if task['name'] == nested_custom_task['root_ct']:
                        task['params'].extend(copy.deepcopy(nested_custom_task_special_params))
                    elif task['name'] in nested_custom_task['ancestors'] or task[
                        'name'] == nested_custom_task['father_ct']:
                        task['params'].extend(nested_custom_task_special_params)
                    if task.get('params') is not None:
                        task['params'] = sorted(task['params'], key=lambda k: k['name'])
                for special_param in nested_custom_task_special_params:
                    for nested_param in nested_custom_task_spec['params']:
                        if nested_param['name'] == special_param['name']:
                            nested_param['value'] = '$(params.%s)' % nested_param['name']
                # need process parameters to replace results
                custom_task_cr_task_names = [cr_task['name'] for cr_task in custom_task_cr['spec']['pipelineSpec']['tasks']]
                for nested_custom_task_param in nested_custom_task_spec['params']:
                    if '$(tasks.' in nested_custom_task_param['value']:
                        param_results = re.findall('\$\(tasks.([^ \t\n.:,;\{\}]+).results.([^ \t\n.:,;\{\}]+)\)',
                                                    nested_custom_task_param['value'])
                        for param_result in param_results:
                            if param_result[0] not in custom_task_cr_task_names:
                                custom_task_cr_param_names = [p['name'] for p in custom_task_cr['spec']['pipelineSpec']['params']]
                                if nested_custom_task_param['name'] not in custom_task_cr_param_names:
                                    for index, param in enumerate(nested_custom_task_spec['params']):
                                        if nested_custom_task_param['name'] == param['name']:
                                            nested_custom_task_spec['params'].pop(index)
                                            break
                                else:
                                    nested_custom_task_spec = json.loads(
                                        json.dumps(nested_custom_task_spec).replace(nested_custom_task_param['value'],
                                        '$(params.%s)' % nested_custom_task_param['name']))
                # add nested custom task spec to main custom task
                custom_task_cr['spec']['pipelineSpec']['tasks'].append(nested_custom_task_spec)
                custom_task_cr['spec']['pipelineSpec']['params'] = sorted(
                    custom_task_cr['spec']['pipelineSpec']['params'], key=lambda k: k['name'])

    # remove the tasks belong to custom task from main workflow
    task_name_prefix = '-'.join(group_names[:-1] + [""])
    for task in tasks:
        if task['name'].replace(task_name_prefix, "") not in task_list:
            task_list_trimmed = [sanitize_k8s_name(task, max_length=LOOP_GROUP_NAME_LENGTH, rev_truncate=True) for task in task_list]
            if task['name'].replace(task_name_prefix, "") not in task_list_trimmed:
                new_tasks.append(task)
    workflow['spec']['pipelineSpec']['tasks'] = new_tasks
    return custom_task_crs, workflow


def find_ancestors(nested_custom_tasks: list, father_ct_name, ancestors: list, root_ct):
    relationship = {'ancestors': ancestors, 'root_ct': root_ct}
    for custom_task in nested_custom_tasks:
        if father_ct_name == custom_task['nested_custom_task']:
            father_ct_name = custom_task['father_ct']
            relationship = find_ancestors(nested_custom_tasks, father_ct_name, ancestors, father_ct_name)
            if relationship['root_ct'] != father_ct_name:
                ancestors.append(father_ct_name)
    return relationship
