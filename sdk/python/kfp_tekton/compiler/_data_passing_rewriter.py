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

from typing import List, Optional, Set

from kfp_tekton.compiler._k8s_helper import sanitize_k8s_name
from kfp_tekton.compiler._op_to_template import _get_base_step, _add_mount_path, _prepend_steps
from os import environ as env


def fix_big_data_passing(workflow: dict) -> dict:
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
            param.get('name') for param in task["taskSpec"].get('params', [])
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

    pipelinerun_template = workflow

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
        for input_artifact in template['taskSpec'].get('artifacts', {}):
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
        task_template = big_data_passing_tasks(task_template,
                                               pipelinerun_template,
                                               inputs_consumed_as_artifacts,
                                               outputs_consumed_as_artifacts)

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
        task['params'] = [
            parameter_argument
            for parameter_argument in task.get('params', [])
            if (task['name'], parameter_argument['name']
                ) in inputs_consumed_as_parameters and
            (task['name'],
             parameter_argument['name']) not in inputs_consumed_as_artifacts
            or task['name'] in resource_template_names
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
                # Add workspaces instead of parmas, for tasks of big data inputs
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
        pr['spec']['workspaces'].append({
            "name": pipelinerun_name,
            "volumeClaimTemplate": {
                "spec": {
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


def big_data_passing_tasks(task: dict, pipelinerun_template: dict,
                           inputs_tasks: set, outputs_tasks: set) -> dict:
    task_name = task.get('name')
    task_spec = task.get('taskSpec', {})
    # Data passing for the task outputs
    task_outputs = task_spec.get('results', [])
    for task_output in task_outputs:
        if (task_name, task_output.get('name')) in outputs_tasks:
            if not task_spec.setdefault('workspaces', []):
                task_spec['workspaces'].append({"name": task_name})
            # Replace the args for the outputs in the task_spec
            # $(results.task_output.get('name').path)  -->
            # $(workspaces.task_name.path)/task_name-task_output.get('name')
            placeholder = '$(results.%s.path)' % (sanitize_k8s_name(
                task_output.get('name')))
            workspaces_parameter = '$(workspaces.%s.path)/%s-%s' % (
                task_name, task_name, task_output.get('name'))
            task['taskSpec'] = replace_big_data_placeholder(
                task['taskSpec'], placeholder, workspaces_parameter)
            pipelinerun_template['metadata']['annotations'] = replace_big_data_placeholder(
                pipelinerun_template['metadata']['annotations'], placeholder, workspaces_parameter)

    # Remove artifacts outputs from results
    task['taskSpec']['results'] = [
        result for result in task_outputs
        if (task_name, result.get('name')) not in outputs_tasks
    ]

    # Data passing for task inputs
    task_spec = task.get('taskSpec', {})
    task_parmas = task_spec.get('params', [])
    task_artifacts = task_spec.get('artifacts', [])
    for task_parma in task_parmas:
        if (task_name, task_parma.get('name')) in inputs_tasks:
            if not task_spec.setdefault('workspaces', []):
                task_spec['workspaces'].append({"name": task_name})
            # Replace the args for the inputs in the task_spec
            # /tmp/inputs/text/data ---->
            # $(workspaces.task_name.path)/task_parma.get('name')
            placeholder = '/tmp/inputs/text/data'
            for task_artifact in task_artifacts:
                if task_artifact.get('name') == task_parma.get('name'):
                    placeholder = task_artifact.get('path')
            workspaces_parameter = '$(workspaces.%s.path)/%s' % (
                task_name, task_parma.get('name'))
            task['taskSpec'] = replace_big_data_placeholder(
                task_spec, placeholder, workspaces_parameter)
    # Handle the case of input artifact without dependent the output of other tasks
    for task_artifact in task_artifacts:
        if (task_name, task_artifact.get('name')) not in inputs_tasks:
            # add input artifact processes
            task = input_artifacts_tasks(task, task_artifact)

    # Remove artifacts parameter from params
    task['taskSpec']['params'] = [
        param for param in task_spec.get('params', [])
        if (task_name, param.get('name')) not in inputs_tasks
    ]

    # Remove artifacts from task_spec
    if 'artifacts' in task_spec:
        del task['taskSpec']['artifacts']

    return task


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
