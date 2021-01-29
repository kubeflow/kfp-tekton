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

import json


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
        for key, val in pipeline_variables.items():
            task_str = json.dumps(task['taskSpec']['steps'])
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


def _handle_tekton_custom_task(custom_task: dict, workflow: dict):
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
                    if dep_task not in custom_task[dependency['runAfter']]['task_list']:
                        task_dependencies.append(dep_task)
                task['runAfter'] = task_dependencies
    # get custom tasks
    for custom_task_key in custom_task.keys():
        denpendency_list = custom_task[custom_task_key]['spec'].get('runAfter', [])
        task_list.extend(custom_task[custom_task_key]['task_list'])
        # generate custom task cr
        custom_task_cr_tasks = []
        custom_task_cr_params = []
        for task in tasks:
            if task['name'] in custom_task[custom_task_key]['task_list']:
                for param in task.get('taskSpec', {}).get('params', []):
                    param['type'] = 'string'
                run_after_task_list = []
                for run_after_task in task.get('runAfter', []):
                    if run_after_task not in denpendency_list:
                        run_after_task_list.append(run_after_task)
                if task.get('runAfter', []):
                    task['runAfter'] = run_after_task_list
                custom_task_cr_tasks.append(task)
                for param in task.get('params', []):
                    if '$(params.' in param['value'] and param not in custom_task_cr_params:
                        custom_task_cr_params.append(param)
                    if '$(tasks.' in param['value'] and param not in custom_task_cr_params:
                        custom_task_cr_params.append(param)
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
                    } for parm in custom_task_cr_params],
                    "tasks": custom_task_cr_tasks
                }
            }
        }
        # handle loop special case
        if custom_task[custom_task_key]['kind'] == 'loops':
            custom_task_cr['kind'] = 'PipelineLoop'
            custom_task_cr['spec']['iterateParam'] = custom_task[custom_task_key]['loop_args']
            for param in custom_task_cr_params:
                if param['name'] != custom_task[custom_task_key]['loop_args']:
                    custom_task[custom_task_key]['spec']['params'].append(param)

        custom_task_crs.append(custom_task_cr)
        tasks.append(custom_task[custom_task_key]['spec'])

    # handle the nested custom task case
    nested_custom_tasks = []
    custom_task_crs_namelist = [custom_task_key for custom_task_key in custom_task.keys()]
    for custom_task_key in custom_task.keys():
        for inner_task_name in custom_task[custom_task_key]['task_list']:
            if inner_task_name in custom_task_crs_namelist:
                nested_custom_tasks.append({"main_ct": custom_task_key, "nested_custom_task": inner_task_name})
    for nested_custom_task in nested_custom_tasks:
        for custom_task_cr in custom_task_crs:
            if custom_task_cr['metadata']['name'] == nested_custom_task['main_ct']:
                custom_task_cr['spec']['pipelineSpec']['tasks'].append(custom_task[nested_custom_task['nested_custom_task']]['spec'])

    # remove the tasks belong to custom task from main workflow
    for task in tasks:
        if task['name'] not in task_list:
            new_tasks.append(task)
    workflow['spec']['pipelineSpec']['tasks'] = new_tasks
    return custom_task_crs, workflow
