# Copyright 2019-2020 kubeflow.org
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

import copy
import json
import re
import pathlib

from typing import List, Optional, Set

from kfp_tekton.compiler._k8s_helper import sanitize_k8s_name
from kfp_tekton.compiler._op_to_template import _get_base_step, _add_mount_path, _prepend_steps
from os import environ as env

BIG_DATA_MIDPATH = "artifacts/$ORIG_PR_NAME"
BIG_DATA_PATH_FORMAT = "/".join(["$(workspaces.$TASK_NAME.path)", BIG_DATA_MIDPATH, "$TASKRUN_NAME", "$TASK_PARAM_NAME"])


def fix_big_data_passing(workflow: dict, loops_pipeline: dict, loop_name_prefix: str) -> dict:
    """
    fix_big_data_passing converts a workflow where some artifact data is passed
    as parameters and converts it to a workflow where this data is passed as
    artifacts.

    Args:
        workflow: The workflow to fix
    Returns:
        The fixed workflow

    Motivation:

    DSL compiler only supports passing Tekton parameters.
    Due to the convoluted nature of the DSL compiler, the artifact consumption
    and passing has been implemented on top of that using parameter passing.
    The artifact data is passed as parameters and the consumer template creates
    an artifact/file out of that data.
    Due to the limitations of Kubernetes and Tekton this scheme cannot pass data
    larger than few kilobytes preventing any serious use of artifacts.

    This function rewrites the compiled workflow so that the data consumed as
    artifact is passed as artifact.
    It also prunes the unused parameter outputs. This is important since if a
    big piece of data is ever returned through a file that is also output as
    parameter, the execution will fail.
    This makes is possible to pass large amounts of data.

    Implementation:

    1. Index the pipelines to understand how data is being passed and which inputs/outputs
       are connected to each other.
    2. Search for direct data consumers in container/resource templates and some
       pipeline task attributes (e.g. conditions and loops) to find out which inputs
       are directly consumed as parameters/artifacts.
    3. Propagate the consumption information upstream to all inputs/outputs all
       the way up to the data producers.
    4. Convert the inputs, outputs based on how they're consumed downstream.
    5. Use workspaces instead of result and params for big data passing.
    6. Added workspaces to tasks, pipelines, pipelineruns, if the parmas is big data.
    7. A PVC named with pipelinerun name will be created if big data is passed, as workspaces need to use it.
       User need to define proper volume or enable dynamic volume provisioning refer to the link of:
       https://kubernetes.io/docs/concepts/storage/dynamic-provisioning
    """

    workflow = copy.deepcopy(workflow)

    tasks = workflow["spec"]["pipelineSpec"].get(
        'tasks', []) + workflow["spec"]["pipelineSpec"].get('finally', [])

    resource_templates = []
    for task in tasks:
        resource_params = [
            param.get('name') for param in task.get("taskSpec", {}).get('params', [])
            if param.get('name') == 'action'
            or param.get('name') == 'success-condition'
        ]
        if 'action' in resource_params and 'success-condition' in resource_params:
            resource_templates.append(task)

    # TODO: use split on condition
    resource_template_names = set(task["name"] for task in resource_templates)

    container_templates = [
        task for task in tasks if task["name"] not in resource_template_names
    ]

    pipeline_template = workflow["spec"]["pipelineSpec"]

    pipelinerun_template = load_annotations(workflow)

    # 1. Index the pipelines to understand how data is being passed and which
    #  inputs/outputs are connected to each other.
    template_input_to_parent_pipeline_inputs = {
    }  # (task_template_name, task_input_name) -> Set[(pipeline_template_name, pipeline_input_name)]
    template_input_to_parent_task_outputs = {
    }  # (task_template_name, task_input_name) -> Set[(upstream_template_name, upstream_output_name)]
    template_input_to_parent_constant_arguments = {
    }  # (task_template_name, task_input_name) -> Set[argument_value] # Unused
    pipeline_output_to_parent_template_outputs = {
    }  # (pipeline_template_name, output_name) -> Set[(upstream_template_name, upstream_output_name)]

    # pipeline has no name when embedded in pipelineRun
    pipelinerun_name = workflow.get('metadata', {}).get('name')

    # Indexing task arguments
    pipeline_tasks = pipeline_template.get(
        "tasks", []) + pipeline_template.get('finally', [])

    for task in pipeline_tasks:
        task_template_name = task['name']
        parameter_arguments = task['params']
        for parameter_argument in parameter_arguments:
            task_input_name = parameter_argument['name']
            argument_value = parameter_argument['value']

            argument_placeholder_parts = deconstruct_tekton_single_placeholder(
                argument_value)
            if not argument_placeholder_parts:  # Argument is considered to be constant string
                template_input_to_parent_constant_arguments.setdefault(
                    (task_template_name, task_input_name),
                    set()).add(argument_value)
            else:
                placeholder_type = argument_placeholder_parts[0]
                if placeholder_type not in ('params', 'outputs', 'tasks',
                                            'steps', 'workflow', 'pod',
                                            'item'):
                    # Do not fail on Jinja or other double-curly-brace templates
                    continue
                if placeholder_type == 'params':
                    pipeline_input_name = argument_placeholder_parts[1]
                    template_input_to_parent_pipeline_inputs.setdefault(
                        (task_template_name, task_input_name), set()).add(
                            (pipelinerun_name, pipeline_input_name))
                elif placeholder_type == 'tasks':
                    upstream_task_name = argument_placeholder_parts[1]
                    assert argument_placeholder_parts[2] == 'results'
                    upstream_output_name = argument_placeholder_parts[3]
                    # upstream_template_name = upstream_task_name
                    template_input_to_parent_task_outputs.setdefault(
                        (task_template_name, task_input_name), set()).add(
                            (upstream_task_name, upstream_output_name))
                elif placeholder_type in ('item', 'workflow', 'pod'):
                    # workflow.parameters.* placeholders are not supported,
                    # but the DSL compiler does not produce those.
                    template_input_to_parent_constant_arguments.setdefault(
                        (task_template_name, task_input_name),
                        set()).add(argument_value)
                else:
                    raise AssertionError
            pipeline_input_name = extract_tekton_input_parameter_name(
                argument_value)
            if pipeline_input_name:
                template_input_to_parent_pipeline_inputs.setdefault(
                    (task_template_name, task_input_name), set()).add(
                        (pipelinerun_name, pipeline_input_name))
            else:
                template_input_to_parent_constant_arguments.setdefault(
                    (task_template_name, task_input_name),
                    set()).add(argument_value)
    # Finshed indexing the pipelines

    # 2. Search for direct data consumers in container/resource templates and some pipeline task attributes
    #  (e.g. conditions and loops) to find out which inputs are directly consumed as parameters/artifacts.
    inputs_directly_consumed_as_parameters = set()
    inputs_directly_consumed_as_artifacts = set()
    outputs_directly_consumed_as_parameters = set()

    # Searching for artifact input consumers in container template inputs
    for template in container_templates:
        template_name = template['name']
        for input_artifact in template.get("taskSpec", {}).get('artifacts', {}):
            raw_data = input_artifact['raw'][
                'data']  # The structure must exist
            # The raw data must be a single input parameter reference. Otherwise (e.g. it's a string
            #  or a string with multiple inputs) we should not do the conversion to artifact passing.
            input_name = extract_tekton_input_parameter_name(raw_data)
            if input_name:
                inputs_directly_consumed_as_artifacts.add(
                    (template_name, input_name))
                del input_artifact[
                    'raw']  # Deleting the "default value based" data passing hack
                # so that it's replaced by the "argument based" way of data passing.
                input_artifact[
                    'name'] = input_name  # The input artifact name should be the same
                # as the original input parameter name

    # Searching for parameter input consumers in container and resource templates
    for template in container_templates + resource_templates:
        template_name = template['name']
        placeholders = extract_all_tekton_placeholders(
            template.get('taskSpec', {}))
        for placeholder in placeholders:
            parts = placeholder.split('.')
            placeholder_type = parts[0]
            if placeholder_type not in ('inputs', 'outputs', 'tasks', 'steps',
                                        'workflow', 'pod', 'item', 'results'):
                # Do not fail on Jinja or other double-curly-brace templates
                continue

            if placeholder_type == 'workflow' or placeholder_type == 'pod':
                pass
            elif placeholder_type == 'inputs':
                if parts[1] == 'params':
                    input_name = parts[2]
                    inputs_directly_consumed_as_parameters.add(
                        (template_name, input_name))
                elif parts[1] == 'artifacts':
                    raise RuntimeError(
                        'Found unexpected Tekton input artifact placeholder in container template: {}'
                        .format(placeholder))
                else:
                    raise RuntimeError(
                        'Found unexpected Tekton input placeholder in container template: {}'
                        .format(placeholder))
            elif placeholder_type == 'results':
                input_name = parts[1]
                outputs_directly_consumed_as_parameters.add(
                    (template_name, input_name))
            else:
                raise RuntimeError(
                    'Found unexpected Tekton placeholder in container template: {}'
                    .format(placeholder))

    # Finished indexing data consumers

    # 3. Propagate the consumption information upstream to all inputs/outputs all the way up to the data producers.
    inputs_consumed_as_parameters = set()
    inputs_consumed_as_artifacts = set()

    outputs_consumed_as_parameters = set()
    outputs_consumed_as_artifacts = set()

    def mark_upstream_ios_of_input(template_input, marked_inputs,
                                   marked_outputs):
        # Stopping if the input has already been visited to save time and handle recursive calls
        if template_input in marked_inputs:
            return
        marked_inputs.add(template_input)

        upstream_inputs = template_input_to_parent_pipeline_inputs.get(
            template_input, [])
        for upstream_input in upstream_inputs:
            mark_upstream_ios_of_input(upstream_input, marked_inputs,
                                       marked_outputs)

        upstream_outputs = template_input_to_parent_task_outputs.get(
            template_input, [])
        for upstream_output in upstream_outputs:
            mark_upstream_ios_of_output(upstream_output, marked_inputs,
                                        marked_outputs)

    def mark_upstream_ios_of_output(template_output, marked_inputs,
                                    marked_outputs):
        # Stopping if the output has already been visited to save time and handle recursive calls
        if template_output in marked_outputs:
            return
        marked_outputs.add(template_output)

        upstream_outputs = pipeline_output_to_parent_template_outputs.get(
            template_output, [])
        for upstream_output in upstream_outputs:
            mark_upstream_ios_of_output(upstream_output, marked_inputs,
                                        marked_outputs)

    for input in inputs_directly_consumed_as_parameters:
        mark_upstream_ios_of_input(input, inputs_consumed_as_parameters,
                                   outputs_consumed_as_parameters)
    for input in inputs_directly_consumed_as_artifacts:
        mark_upstream_ios_of_input(input, inputs_consumed_as_artifacts,
                                   outputs_consumed_as_artifacts)
    for output in outputs_directly_consumed_as_parameters:
        mark_upstream_ios_of_output(output, inputs_consumed_as_parameters,
                                    outputs_consumed_as_parameters)

    # 4. Convert the inputs, outputs and arguments based on how they're consumed downstream.
    # Add workspaces to pipeline and pipeline task_ref if big data passing
    pipeline_workspaces = set()
    pipelinerun_workspaces = set()
    output_tasks_consumed_as_artifacts = {
        output[0]
        for output in outputs_consumed_as_artifacts
    }
    # task_workspaces = set()

    # Converting pipeline inputs
    pipeline, pipeline_workspaces = big_data_passing_pipeline(
        pipelinerun_name, pipeline_template, inputs_consumed_as_artifacts,
        output_tasks_consumed_as_artifacts)

    # Add workspaces to pipelinerun if big data passing
    if pipeline_workspaces:
        pipeline, pipelinerun_workspaces = big_data_passing_pipelinerun(
            pipelinerun_name, pipelinerun_template, pipeline_workspaces)

    # Use workspaces to tasks if big data passing instead of 'results', 'copy-inputs'
    for task_template in container_templates:
        task_template = big_data_passing_tasks(pipelinerun_name,
                                               task_template,
                                               pipelinerun_template,
                                               inputs_consumed_as_artifacts,
                                               outputs_consumed_as_artifacts,
                                               loops_pipeline,
                                               loop_name_prefix)

    # Remove input parameters unless they're used downstream.
    # This also removes unused container template inputs if any.
    for template in container_templates + [pipeline_template]:
        spec = template.get('taskSpec', {}) or template.get('pipelineSpec', {})
        spec['params'] = [
            input_parameter for input_parameter in spec.get('params', []) if (
                template.get(
                    'name'
                ),  # TODO: pipeline has no name, use pipelineRun name?
                input_parameter['name']) in inputs_consumed_as_parameters
                or input_parameter['name'].endswith("-trname")
        ]

    # Remove output parameters unless they're used downstream
    for template in container_templates + [pipeline_template]:
        spec = template.get('taskSpec', {}) or template.get('pipelineSpec', {})
        spec['results'] = [
            output_parameter for output_parameter in spec.get('results', [])
        ]
        # tekton results doesn't support underscore
        renamed_results_in_pipeline_task = set()
        for task_result in spec['results']:
            task_result_old_name = task_result.get('name')
            task_result_new_name = sanitize_k8s_name(task_result_old_name)
            if task_result_new_name != task_result_old_name:
                task_result['name'] = task_result_new_name
                renamed_results_in_pipeline_task.add(
                    (task_result_old_name, task_result_new_name))
        for renamed_result in renamed_results_in_pipeline_task:
            # Change results.downloaded_resultOutput to results.downloaded-resultoutput
            template['taskSpec'] = replace_big_data_placeholder(
                spec, 'results.%s' % renamed_result[0],
                'results.%s' % renamed_result[1])

    # Remove pipeline task parameters unless they're used downstream
    for task in pipeline_tasks:
        # Don't process condition and custom task parameters
        is_condition = 'condition-' in task['name']
        is_custom_task = task.get('taskRef') or task.get('taskSpec', {}).get('apiVersion')
        if not is_condition and not is_custom_task:
            task['params'] = [
                parameter_argument
                for parameter_argument in task.get('params', [])
                if (task['name'], parameter_argument['name']
                    ) in inputs_consumed_as_parameters and
                (task['name'],
                parameter_argument['name']) not in inputs_consumed_as_artifacts
                or task['name'] in resource_template_names
                or parameter_argument['name'].endswith("-trname")
            ]

        # tekton results doesn't support underscore
        for argument in task['params']:
            argument_value = argument.get('value')
            argument_placeholder_parts = deconstruct_tekton_single_placeholder(
                argument_value)
            if len(argument_placeholder_parts) == 4 \
                    and argument_placeholder_parts[0] == 'tasks':
                argument['value'] = '$(tasks.%s.%s.%s)' % (
                    argument_placeholder_parts[1],
                    argument_placeholder_parts[2],
                    sanitize_k8s_name(argument_placeholder_parts[3]))

    workflow = jsonify_annotations(workflow)
    # Need to confirm:
    # I didn't find the use cases to support workflow parameter consumed as artifacts downstream in tekton.
    # Whether this case need to be supporting?
    clean_up_empty_workflow_structures(workflow)
    return workflow


def extract_all_tekton_placeholders(template: dict) -> Set[str]:
    template_str = json.dumps(template)
    placeholders = set(re.findall('\\$\\(([-._a-zA-Z0-9]+)\\)', template_str))
    return placeholders


def extract_tekton_input_parameter_name(s: str) -> Optional[str]:
    match = re.fullmatch('\\$\\(inputs.params.([-_a-zA-Z0-9]+)\\)', s)
    if not match:
        return None
    (input_name, ) = match.groups()
    return input_name


def deconstruct_tekton_single_placeholder(s: str) -> List[str]:
    if not re.fullmatch('\\$\\([-._a-zA-Z0-9]+\\)', s):
        return []
    return s.lstrip('$(').rstrip(')').split('.')


def replace_big_data_placeholder(template: dict, old_str: str,
                                 new_str: str) -> dict:
    template_str = json.dumps(template)
    template_str = template_str.replace(old_str, new_str)
    template = json.loads(template_str)
    return template


def big_data_passing_pipeline(name: str, template: dict, inputs_tasks: set(),
                              outputs_tasks: set):
    pipeline_workspaces = set()
    pipeline_name = name
    pipeline_spec = template
    tasks = pipeline_spec.get('tasks', []) + pipeline_spec.get('finally', [])
    for task in tasks:
        parameter_arguments = task.get('params', [])
        for parameter_argument in parameter_arguments:
            input_name = parameter_argument['name']
            if (task.get('name'), input_name) in inputs_tasks:
                pipeline_workspaces.add(pipeline_name)
                # Add workspaces instead of params, for tasks of big data inputs
                if not task.setdefault('workspaces', []):
                    task['workspaces'].append({
                        "name": task.get('name'),
                        "workspace": pipeline_name
                    })
                # runAfter add to the task, which was depends on the result of the task.
                argument_value = parameter_argument['value']
                argument_placeholder_parts = deconstruct_tekton_single_placeholder(
                    argument_value)
                if argument_placeholder_parts[
                        0] == 'tasks' and argument_placeholder_parts[
                            2] == 'results':
                    dependency_task = argument_placeholder_parts[1]
                    task.setdefault('runAfter', [])
                    task['runAfter'].append(dependency_task)
        if task.get('name') in outputs_tasks:
            # Add workspaces for tasks of big data outputs
            if not task.setdefault('workspaces', []):
                task['workspaces'].append({
                    "name": task.get('name'),
                    "workspace": pipeline_name
                })
    if pipeline_name in pipeline_workspaces:
        # Add workspaces to pipeline
        if not pipeline_spec.setdefault('workspaces', []):
            pipeline_spec['workspaces'].append({"name": pipeline_name})
    return template, pipeline_workspaces


def big_data_passing_pipelinerun(name: str, pr: dict, pw: set):
    prw = set()
    pipelinerun_name = name
    if pipelinerun_name in pw:
        pr.get('spec', {}).setdefault('workspaces', [])
        # Change persistentVolumeClaim to volumeClaimTemplate which need Tekton version > 0.12
        # User could modify default size of storage in the yaml file mannually if necessary.
        DEFAULT_ACCESSMODES = env.get('DEFAULT_ACCESSMODES', 'ReadWriteMany')
        DEFAULT_STORAGE_SIZE = env.get('DEFAULT_STORAGE_SIZE', '2Gi')
        DEFAULT_STORAGE_CLASS = env.get('DEFAULT_STORAGE_CLASS', 'kfp-csi-s3')
        pr['spec']['workspaces'].append({
            "name": pipelinerun_name,
            "volumeClaimTemplate": {
                "spec": {
                    "storageClassName": DEFAULT_STORAGE_CLASS,
                    "accessModes": [DEFAULT_ACCESSMODES],
                    "resources": {
                        "requests": {
                            "storage": DEFAULT_STORAGE_SIZE
                        }
                    }
                }
            }
        })
        prw.add(pipelinerun_name)
    return pr, prw


def big_data_passing_tasks(prname: str, task: dict, pipelinerun_template: dict,
                            inputs_tasks: set, outputs_tasks: set, loops_pipeline: dict,
                            loop_name_prefix: str) -> dict:
    task_name = task.get('name')
    task_spec = task.get('taskSpec', {})
    # Data passing for the task outputs
    appended_taskrun_name = False
    for task_output in task.get('taskSpec', {}).get('results', []):
        if (task_name, task_output.get('name')) in outputs_tasks:
            if not task.get('taskSpec', {}).setdefault('workspaces', []):
                task.get('taskSpec', {})['workspaces'].append({"name": task_name})
            # Replace the args for the outputs in the task_spec
            # $(results.task_output.get('name').path)  -->
            # $(workspaces.task_name.path)/task_name-task_output.get('name')
            placeholder = '$(results.%s.path)' % (sanitize_k8s_name(
                task_output.get('name')))
            workspaces_parameter = '$(workspaces.%s.path)/%s/%s/%s' % (
                task_name, BIG_DATA_MIDPATH, "$(context.taskRun.name)", task_output.get('name'))
            # For child nodes to know the taskrun name, it has to pass to results via /tekton/results emptydir
            if not appended_taskrun_name:
                copy_taskrun_name_step = _get_base_step('output-taskrun-name')
                copy_taskrun_name_step['script'] += 'echo -n "%s" > $(results.taskrun-name.path)\n' % ("$(context.taskRun.name)")
                task['taskSpec']['results'].append({"name": "taskrun-name"})
                task['taskSpec']['steps'].append(copy_taskrun_name_step)
                _append_original_pr_name_env(task)
                appended_taskrun_name = True
            task['taskSpec'] = replace_big_data_placeholder(
                task.get("taskSpec", {}), placeholder, workspaces_parameter)
            artifact_items = pipelinerun_template['metadata']['annotations']['tekton.dev/artifact_items']
            artifact_items[task['name']] = replace_big_data_placeholder(
                artifact_items[task['name']], placeholder, workspaces_parameter)
            pipelinerun_template['metadata']['annotations']['tekton.dev/artifact_items'] = \
                artifact_items

    task_spec = task.get('taskSpec', {})
    task_params = task_spec.get('params', [])
    task_artifacts = task_spec.get('artifacts', [])

    # Data passing for task inputs
    for task_param in task_params:
        if (task_name, task_param.get('name')) in inputs_tasks:
            if not task_spec.setdefault('workspaces', []):
                task_spec['workspaces'].append({"name": task_name})
            # Replace the args for the inputs in the task_spec
            # /tmp/inputs/text/data ---->
            # $(workspaces.task_name.path)/task_param.get('name')
            placeholder = '/tmp/inputs/text/data'
            for task_artifact in task_artifacts:
                if task_artifact.get('name') == task_param.get('name'):
                    placeholder = task_artifact.get('path')
            task_param_task_name = ""
            task_param_param_name = ""
            for o_task in outputs_tasks:
                if '-'.join(o_task) == task_param.get('name'):
                    task_param_task_name = o_task[0]
                    task_param_param_name = o_task[1]
                    break
            # If the param name is constructed with task_name-param_name,
            # use the current task_name as the path prefix

            def append_taskrun_params(task_name_append: str):
                taskrun_param_name = task_name_append + "-trname"
                inserted_taskrun_param = False
                for param in task['taskSpec'].get('params', []):
                    if param.get('name', "") == taskrun_param_name:
                        inserted_taskrun_param = True
                        break
                if not inserted_taskrun_param:
                    task['taskSpec']['params'].append({"name": taskrun_param_name})
                    task['params'].append({"name": taskrun_param_name, "value": "$(tasks.%s.results.taskrun-name)" % task_name_append})
                    parent_task_queue = [task['name']]
                    while parent_task_queue:
                        current_task = parent_task_queue.pop(0)
                        for loop_name, loop_spec in loops_pipeline.items():
                            # print(loop_name, loop_spec)
                            if current_task in loop_spec.get('task_list', []):
                                parent_task_queue.append(loop_name.replace(loop_name_prefix, ""))
                                loop_param_names = [loop_param['name'] for loop_param in loops_pipeline[loop_name]['spec']['params']]
                                if task_name_append + '-taskrun-name' in loop_param_names:
                                    continue
                                loops_pipeline[loop_name]['spec']['params'].append({'name': task_name_append + '-taskrun-name',
                                'value': '$(tasks.%s.results.taskrun-name)' % task_name_append})

            if task_param_task_name:
                workspaces_parameter = '$(workspaces.%s.path)/%s/$(params.%s-trname)/%s' % (
                    task_name, BIG_DATA_MIDPATH, task_param_task_name, task_param_param_name)
                if task_param_task_name != task_name:
                    append_taskrun_params(task_param_task_name)  # need to get taskrun name from parent path
            else:
                workspaces_parameter = '$(workspaces.%s.path)/%s/%s/%s' % (
                    task_name, BIG_DATA_MIDPATH, "$(context.taskRun.name)", task_param.get('name'))
            _append_original_pr_name_env(task)
            task['taskSpec'] = replace_big_data_placeholder(
                task_spec, placeholder, workspaces_parameter)
            task_spec = task.get('taskSpec', {})
    # Handle the case of input artifact without dependent the output of other tasks
    for task_artifact in task_artifacts:
        if (task_name, task_artifact.get('name')) not in inputs_tasks:
            # add input artifact processes
            task = input_artifacts_tasks(task, task_artifact)

        if (prname, task_artifact.get('name')) in inputs_tasks:
            # add input artifact processes for pipeline parameter
            if not task_artifact.setdefault('raw', {}):
                for i in range(len(pipelinerun_template['spec']['params'])):
                    param_name = pipelinerun_template['spec']['params'][i]['name']
                    param_value = pipelinerun_template['spec']['params'][i]['value']
                    if (task_artifact.get('name') == param_name):
                        task_artifact['raw']['data'] = param_value
                        task = input_artifacts_tasks_pr_params(task, task_artifact)

    # If a task produces a result and artifact, add a step to copy artifact to results.
    artifact_items = pipelinerun_template['metadata']['annotations']['tekton.dev/artifact_items']
    add_copy_results_artifacts_step = False
    if task.get("taskSpec", {}):
        if task_spec.get('results', []):
            copy_results_artifact_step = _get_base_step('copy-results-artifacts')
            copy_results_artifact_step['onError'] = 'continue'  # supported by v0.27+ of tekton.
            copy_results_artifact_step['script'] += 'TOTAL_SIZE=0\n'
            for result in task_spec['results']:
                if task['name'] in artifact_items:
                    artifact_i = artifact_items[task['name']]
                    for index, artifact_tuple in enumerate(artifact_i):
                        artifact_name, artifact = artifact_tuple
                        src = artifact
                        dst = '$(results.%s.path)' % sanitize_k8s_name(result['name'])
                        if artifact_name == result['name'] and src != dst:
                            add_copy_results_artifacts_step = True
                            copy_results_artifact_step['script'] += (
                                    'ARTIFACT_SIZE=`wc -c %s | awk \'{print $1}\'`\n' % src +
                                    'TOTAL_SIZE=$( expr $TOTAL_SIZE + $ARTIFACT_SIZE)\n' +
                                    'touch ' + dst + '\n' +  # create an empty file by default.
                                    'if [[ $TOTAL_SIZE -lt 3072 ]]; then\n' +
                                    '  cp ' + src + ' ' + dst + '\n' +
                                    'fi\n'
                            )
            _append_original_pr_name_env_to_step(copy_results_artifact_step)
            if add_copy_results_artifacts_step:
                task['taskSpec']['steps'].append(copy_results_artifact_step)

    # Remove artifacts parameter from params
    task.get("taskSpec", {})['params'] = [
        param for param in task_spec.get('params', [])
        if (task_name, param.get('name')) not in inputs_tasks or
            param.get('name').endswith("-trname")
    ]

    # Remove artifacts from task_spec
    if 'artifacts' in task_spec:
        del task['taskSpec']['artifacts']

    return task


def input_artifacts_tasks_pr_params(template: dict, artifact: dict) -> dict:
    copy_inputs_step = _get_base_step('copy-inputs')
    task_name = template.get('name')
    task_spec = template.get('taskSpec', {})
    task_params = task_spec.get('params', [])
    for task_param in task_params:
        # For pipeline parameter input artifacts, it will never come from another task because pipeline
        # params are global parameters. Thus, task_name will always be the executing task name.
        workspaces_parameter = '$(workspaces.%s.path)/%s/%s/%s' % (
            task_name, BIG_DATA_MIDPATH, "$(context.taskRun.name)", task_param.get('name'))
        if 'raw' in artifact:
            copy_inputs_step['script'] += 'mkdir -p %s\n' % pathlib.Path(workspaces_parameter).parent
            copy_inputs_step['script'] += 'echo -n "%s" > %s\n' % (
                artifact['raw']['data'], workspaces_parameter)
        _append_original_pr_name_env(template)

    template['taskSpec']['steps'] = _prepend_steps(
        [copy_inputs_step], template['taskSpec']['steps'])

    return template


def input_artifacts_tasks(template: dict, artifact: dict) -> dict:
    # The input artifacts in KFP is not pulling from s3, it will always be passed as a raw input.
    # Visit https://github.com/kubeflow/pipelines/issues/336 for more details on the implementation.
    volume_mount_step_template = []
    volume_template = []
    mounted_param_paths = []
    copy_inputs_step = _get_base_step('copy-inputs')
    if 'raw' in artifact:
        copy_inputs_step['script'] += 'echo -n "%s" > %s\n' % (
            artifact['raw']['data'], artifact['path'])
    mount_path = artifact['path'].rsplit("/", 1)[0]
    if mount_path not in mounted_param_paths:
        _add_mount_path(artifact['name'], artifact['path'], mount_path,
                        volume_mount_step_template, volume_template,
                        mounted_param_paths)
    template['taskSpec']['steps'] = _prepend_steps(
        [copy_inputs_step], template['taskSpec']['steps'])
    # _update_volumes(template, volume_mount_step_template, volume_template)
    if volume_mount_step_template:
        template['taskSpec']['stepTemplate'] = {}
        template['taskSpec']['stepTemplate']['volumeMounts'] = volume_mount_step_template
        template['taskSpec']['volumes'] = volume_template
    return template


def clean_up_empty_workflow_structures(workflow: list):
    template_spec = workflow['spec']
    if not template_spec.setdefault('params', []):
        del template_spec['params']
    if not template_spec.setdefault('artifacts', []):
        del template_spec['artifacts']
    if not template_spec.setdefault('results', []):
        del template_spec['results']
    if not template_spec:  # ?
        del workflow['spec']
    # if template['kind'] == 'Pipeline':
    for task in workflow['spec']['pipelineSpec']['tasks'] + workflow['spec'][
            'pipelineSpec'].get('finally', []):
        if not task.setdefault('params', []):
            del task['params']
        if not task.setdefault('artifacts', []):
            del task['artifacts']
    if not workflow['spec']['pipelineSpec'].setdefault('finally', []):
        del workflow['spec']['pipelineSpec']['finally']


def load_annotations(template: dict):
    artifact_items = json.loads(
        str(template['metadata']['annotations']['tekton.dev/artifact_items']))
    template['metadata']['annotations']['tekton.dev/artifact_items'] = \
        artifact_items
    return template


def jsonify_annotations(template: dict):
    template['metadata']['annotations']['tekton.dev/artifact_items'] = \
        json.dumps(template['metadata']['annotations']['tekton.dev/artifact_items'])
    return template


def _append_original_pr_name_env_to_step(step):
    step.setdefault('env', [])
    has_original_pr_name = False
    for task_env in step['env']:
        if task_env['name'] == "ORIG_PR_NAME":
            has_original_pr_name = True
    if not has_original_pr_name:
        step['env'].append({"name": "ORIG_PR_NAME",
                            "valueFrom":
                                {"fieldRef":
                                    {"fieldPath": "metadata.labels['custom.tekton.dev/originalPipelineRun']"}}})


def _append_original_pr_name_env(task_template):
    for step in task_template['taskSpec']['steps']:
        if step['name'] == 'main':
            _append_original_pr_name_env_to_step(step)
