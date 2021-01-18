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
