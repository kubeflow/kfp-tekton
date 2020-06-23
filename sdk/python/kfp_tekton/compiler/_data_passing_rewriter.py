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

from typing import List, Text, Dict, Any
# from kfp.compiler._data_passing_rewriter import fix_big_data_passing
import copy
import json
import re
from typing import Optional, Set
from . import _op_to_template
from kfp_tekton.compiler._k8s_helper import sanitize_k8s_name


def fix_big_data_passing(
    workflow: List[Dict[Text, Any]]
) -> List[Dict[Text, Any]]:  # Tekton change signature
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
    resource_templates = []
    for template in workflow:
        resource_params = [
            param.get('name')
            for param in template.get('spec', {}).get('params', [])
            if param.get('name') == 'action'
            or param.get('name') == 'success-condition'
        ]
        if 'action' in resource_params and 'success-condition' in resource_params:
            resource_templates.append(template)

    resource_template_names = set(
        template.get('metadata', {}).get('name')
        for template in resource_templates)

    container_templates = [
        template for template in workflow if template['kind'] == 'Task' and
        template.get('metadata', {}).get('name') not in resource_template_names
    ]

    pipeline_templates = [
        template for template in workflow if template['kind'] == 'Pipeline'
    ]

    pipelinerun_templates = [
        template for template in workflow if template['kind'] == 'PipelineRun'
    ]

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

    for template in pipeline_templates:
        pipeline_template_name = template.get('metadata', {}).get('name')
        # Indexing task arguments
        pipeline_tasks = template.get('spec', {}).get('tasks', []) + template.get('spec', {}).get('finally', [])
        task_name_to_template_name = {
            task['name']: task['taskRef']['name']
            for task in pipeline_tasks
        }
        for task in pipeline_tasks:
            task_template_name = task['taskRef']['name']
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
                                (pipeline_template_name, pipeline_input_name))
                    elif placeholder_type == 'tasks':
                        upstream_task_name = argument_placeholder_parts[1]
                        assert argument_placeholder_parts[2] == 'results'
                        upstream_output_name = argument_placeholder_parts[3]
                        upstream_template_name = task_name_to_template_name[
                            upstream_task_name]
                        template_input_to_parent_task_outputs.setdefault(
                            (task_template_name, task_input_name), set()).add(
                                (upstream_template_name, upstream_output_name))
                    elif placeholder_type == 'item' or placeholder_type == 'workflow' or placeholder_type == 'pod':
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
                            (pipeline_template_name, pipeline_input_name))
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
        template_name = template.get('metadata', {}).get('name')
        for input_artifact in template.get('spec', {}).get('artifacts', {}):
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

    # Searching for parameter input consumers in pipeline templates
    # TODO: loop params is not support for tekton yet, refer to https://github.com/kubeflow/kfp-tekton/issues/82
    for template in pipeline_templates:
        template_name = template.get('metadata', {}).get('name')
        pipeline_tasks = template.get('spec', {}).get('tasks', []) + template.get('spec', {}).get('finally', [])
        task_name_to_template_name = {
            task['name']: task['taskRef']['name']
            for task in pipeline_tasks
        }
        for task in pipeline_tasks:
            # We do not care about the inputs mentioned in task arguments
            # since we will be free to switch them from parameters to artifacts
            task_without_arguments = task.copy()  # Shallow copy
            task_without_arguments.pop('params', None)
            placeholders = extract_all_tekton_placeholders(
                task_without_arguments)
            for placeholder in placeholders:
                parts = placeholder.split('.')
                placeholder_type = parts[0]
                if placeholder_type not in ('inputs', 'outputs', 'tasks',
                                            'steps', 'workflow', 'pod',
                                            'item'):
                    # Do not fail on Jinja or other double-curly-brace templates
                    continue
                if placeholder_type == 'inputs':
                    if parts[1] == 'parameters':
                        input_name = parts[2]
                        inputs_directly_consumed_as_parameters.add(
                            (template_name, input_name))
                    else:
                        raise AssertionError
                elif placeholder_type == 'tasks':
                    upstream_task_name = parts[1]
                    assert parts[2] == 'results'
                    upstream_output_name = parts[3]
                    upstream_template_name = task_name_to_template_name[
                        upstream_task_name]
                    outputs_directly_consumed_as_parameters.add(
                        (upstream_template_name, upstream_output_name))
                elif placeholder_type == 'workflow' or placeholder_type == 'pod':
                    pass
                elif placeholder_type == 'item':
                    raise AssertionError(
                        'The "{{item}}" placeholder is not expected outside task arguments.'
                    )
                else:
                    raise AssertionError(
                        'Unexpected placeholder type "{}".'.format(
                            placeholder_type))

    # Searching for parameter input consumers in container and resource templates
    for template in container_templates + resource_templates:
        template_name = template.get('metadata', {}).get('name')
        placeholders = extract_all_tekton_placeholders(template)
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
                    raise AssertionError(
                        'Found unexpected Tekton input artifact placeholder in container template: {}'
                        .format(placeholder))
                else:
                    raise AssertionError(
                        'Found unexpected Tekton input placeholder in container template: {}'
                        .format(placeholder))
            elif placeholder_type == 'results':
                input_name = parts[1]
                outputs_directly_consumed_as_parameters.add(
                    (template_name, input_name))
            else:
                raise AssertionError(
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
    for pipeline in pipeline_templates:
        # Converting pipeline inputs
        pipeline, pipeline_workspaces = big_data_passing_pipeline(
            pipeline, inputs_consumed_as_artifacts,
            output_tasks_consumed_as_artifacts)

    # Add workspaces to pipelinerun if big data passing
    # Check whether pipelinerun was generated, through error if not.
    if pipeline_workspaces:
        if not pipelinerun_templates:
            raise AssertionError(
                'Found big data passing, please enable generate_pipelinerun for your complier'
            )
        for pipelinerun in pipelinerun_templates:
            pipeline, pipelinerun_workspaces = big_data_passing_pipelinerun(
                pipelinerun, pipeline_workspaces)

    # Use workspaces to tasks if big data passing instead of 'results', 'copy-inputs'
    for task_template in container_templates:
        task_template = big_data_passing_tasks(task_template,
                                               inputs_consumed_as_artifacts,
                                               outputs_consumed_as_artifacts)

    # Create pvc for pipelinerun if big data passing.
    # As we used workspaces in tekton pipelines which depends on it.
    # User need to create PV manually, or enable dynamic volume provisioning, refer to the link of:
    # https://kubernetes.io/docs/concepts/storage/dynamic-provisioning
    # TODO: Remove PVC if Tekton version > = 0.12, use 'volumeClaimTemplate' instead
    if pipelinerun_workspaces:
        for pipelinerun in pipelinerun_workspaces:
            workflow.append(create_pvc(pipelinerun))

    # Remove input parameters unless they're used downstream.
    # This also removes unused container template inputs if any.
    for template in container_templates + pipeline_templates:
        spec = template.get('spec', {})
        spec['params'] = [
            input_parameter for input_parameter in spec.get('params', [])
            if (template.get('metadata', {}).get('name'),
                input_parameter['name']) in inputs_consumed_as_parameters
        ]

    # Remove output parameters unless they're used downstream
    for template in container_templates + pipeline_templates:
        spec = template.get('spec', {})
        spec['results'] = [
            output_parameter for output_parameter in spec.get('results', [])
            if (template.get('metadata', {}).get('name'),
                output_parameter['name']) in outputs_consumed_as_parameters
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
            template['spec'] = replace_big_data_placeholder(
                spec, 'results.%s' % renamed_result[0],
                'results.%s' % renamed_result[1])

    # Remove pipeline task parameters unless they're used downstream
    for template in pipeline_templates:
        tasks = template.get('spec', {}).get('tasks', []) + template.get('spec', {}).get('finally', [])
        for task in tasks:
            task['params'] = [
                parameter_argument
                for parameter_argument in task.get('params', [])
                if (task['taskRef']['name'], parameter_argument['name']
                    ) in inputs_consumed_as_parameters and
                (task['taskRef']['name'], parameter_argument['name']
                 ) not in inputs_consumed_as_artifacts
                or task['taskRef']['name'] in resource_template_names
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


def big_data_passing_pipeline(template: dict, inputs_tasks: set(),
                              outputs_tasks: set):
    pipeline_workspaces = set()
    pipeline_name = template.get('metadata', {}).get('name')
    pipeline_spec = template.get('spec', {})
    tasks = pipeline_spec.get('tasks', []) + pipeline_spec.get('finally', [])
    for task in tasks:
        parameter_arguments = task.get('params', [])
        for parameter_argument in parameter_arguments:
            input_name = parameter_argument['name']
            if (task.get('taskRef',
                         {}).get('name'), input_name) in inputs_tasks:
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
        if task.get('taskRef', {}).get('name') in outputs_tasks:
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


def big_data_passing_pipelinerun(pr: dict, pw: set):
    prw = set()
    pipelinerun_name = pr.get('metadata', {}).get('name')
    pipeline_ref_name = pr.get('spec', {}).get('pipelineRef', {}).get('name')
    if pipeline_ref_name in pw:
        pr.get('spec', {}).setdefault('workspaces', [])
        pr['spec']['workspaces'].append({
            "name": pipeline_ref_name,
            "persistentVolumeClaim": {
                "claimName": pipelinerun_name
            }
        })
        prw.add(pipelinerun_name)
    return pr, prw


def big_data_passing_tasks(task: dict, inputs_tasks: set,
                           outputs_tasks: set) -> dict:
    task_name = task.get('metadata', {}).get('name')
    task_spec = task.get('spec', {})
    # Data passing for the task outputs
    task_outputs = task_spec.get('results', [])
    for task_output in task_outputs:
        if (task_name, task_output.get('name')) in outputs_tasks:
            if not task_spec.setdefault('workspaces', []):
                task_spec['workspaces'].append({"name": task_name})
            # Replace the args for the outputs in the task_spec
            # $(results.task_output.get('name').path)  -->
            # $(workspaces.task_name.path)/task_name-task_output.get('name')
            placeholder = '$(results.%s.path)' % (sanitize_k8s_name(task_output.get('name')))
            workspaces_parameter = '$(workspaces.%s.path)/%s-%s' % (
                task_name, task_name, task_output.get('name'))
            task['spec'] = replace_big_data_placeholder(
                task['spec'], placeholder, workspaces_parameter)

    # Remove artifacts outputs from results
    task['spec']['results'] = [
        result for result in task_outputs
        if (task_name, result.get('name')) not in outputs_tasks
    ]

    # Data passing for task inputs
    task_spec = task.get('spec', {})
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
            task['spec'] = replace_big_data_placeholder(
                task_spec, placeholder, workspaces_parameter)
    # Handle the case of input artifact without dependent the output of other tasks
    for task_artifact in task_artifacts:
        if (task_name, task_artifact.get('name')) not in inputs_tasks:
            # add input artifact processes
            task = input_artifacts_tasks(task, task_artifact)

    # Remove artifacts parameter from params
    task['spec']['params'] = [
        parma for parma in task_parmas
        if (task_name, parma.get('name')) not in inputs_tasks
    ]

    # Remove artifacts from task_spec
    if 'artifacts' in task_spec:
        del task['spec']['artifacts']

    return task


# Create pvc for pipelinerun if using big data passing.
# As we used workspaces in tekton pipelines which depends on it.
# User need to create PV manually, or enable dynamic volume provisioning, refer to the link of:
# https://kubernetes.io/docs/concepts/storage/dynamic-provisioning
# TODO: Remove PVC if Tekton version > = 0.12, use 'volumeClaimTemplate' instead
def create_pvc(pr: str) -> dict:
    pvc = {
        "apiVersion": "v1",
        "kind": "PersistentVolumeClaim",
        "metadata": {
            "name": pr
        },
        "spec": {
            "accessModes": ["ReadWriteOnce"],
            "resources": {
                "requests": {
                    "storage": "100Mi"
                }
            },
            "volumeMode": "Filesystem",
        }
    }
    return pvc


def input_artifacts_tasks(template: dict, artifact: dict) -> dict:
    # The input artifacts in KFP is not pulling from s3, it will always be passed as a raw input.
    # Visit https://github.com/kubeflow/pipelines/issues/336 for more details on the implementation.
    volume_mount_step_template = []
    volume_template = []
    mounted_param_paths = []
    copy_inputs_step = _op_to_template._get_base_step('copy-inputs')
    if 'raw' in artifact:
        copy_inputs_step['script'] += 'echo -n "%s" > %s\n' % (
            artifact['raw']['data'], artifact['path'])
    mount_path = artifact['path'].rsplit("/", 1)[0]
    if mount_path not in mounted_param_paths:
        _op_to_template._add_mount_path(artifact['name'], artifact['path'],
                                        mount_path, volume_mount_step_template,
                                        volume_template, mounted_param_paths)
    template['spec']['steps'] = _op_to_template._prepend_steps(
        [copy_inputs_step], template['spec']['steps'])
    _op_to_template._update_volumes(template, volume_mount_step_template,
                                    volume_template)
    return template


def clean_up_empty_workflow_structures(workflow: list):
    for template in workflow:
        template_spec = template.setdefault('spec', {})
        if not template_spec.setdefault('params', []):
            del template_spec['params']
        if not template_spec.setdefault('artifacts', []):
            del template_spec['artifacts']
        if not template_spec.setdefault('results', []):
            del template_spec['results']
        if not template_spec:
            del template['spec']
        if template['kind'] == 'Pipeline':
            for task in template['spec'].get('tasks', []) + template['spec'].get('finally', []):
                if not task.setdefault('params', []):
                    del task['params']
                if not task.setdefault('artifacts', []):
                    del task['artifacts']
            if not template.get('spec', {}).setdefault('finally', []):
                del template.get('spec', {})['finally']
