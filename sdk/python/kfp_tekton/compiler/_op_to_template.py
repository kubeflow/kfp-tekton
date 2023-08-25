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
import re
import yaml
import logging
import hashlib

from collections import OrderedDict
from typing import List, Text, Dict, Any
from os import environ as env

from kfp import dsl
from kfp_tekton.compiler._k8s_helper import convert_k8s_obj_to_json, sanitize_k8s_name
from kfp.compiler._op_to_template import _process_obj, _inputs_to_json, _outputs_to_json
from kfp.dsl._container_op import BaseOp

from kfp_tekton.compiler import __tekton_api_version__ as tekton_api_version


RESOURCE_OP_IMAGE = ":".join(["quay.io/aipipeline/kubectl-wrapper", "latest"])
TEKTON_HOME_RESULT_PATH = "/tekton/home/tep-results/"

# The image to use in basic bash steps such as copying results in multi-step.
TEKTON_BASH_STEP_IMAGE = 'busybox'
TEKTON_COPY_RESULTS_STEP_IMAGE = 'library/bash'
GENERATE_COMPONENT_SPEC_ANNOTATIONS = env.get('GENERATE_COMPONENT_SPEC_ANNOTATIONS', True)


def _get_base_step(name: str, tekton_bash_image_name: str = TEKTON_BASH_STEP_IMAGE):
    """Base image step for running bash commands.

    Return a busybox base step for running bash commands.

    Args:
        name {str}: step name

    Returns:
        Dict[Text, Any]
    """
    return {
        'image': tekton_bash_image_name,
        'name': name,
        'command': ['sh', '-ec']
    }


def _get_copy_result_step_template(step_number: int, result_maps: list):
    """Base copy result step for moving Tekton result files around.

    Return a copy result step for moving Tekton result files around.

    Args:
        step_number {int}: step number
        result_maps {list}: list of maps bucketed with the result groups

    Returns:
        Dict[Text, Any]
    """
    args = [""]
    for key in result_maps[step_number].keys():
        sanitize_key = sanitize_k8s_name(key, allow_capital=True)
        args[0] += "mv %s%s $(results.%s.path);\n" % (TEKTON_HOME_RESULT_PATH, sanitize_key, sanitize_key)
    if step_number > 0:
        for key in result_maps[step_number - 1].keys():
            sanitize_key = sanitize_k8s_name(key, allow_capital=True)
            args[0] += "mv $(results.%s.path) %s%s;\n" % (sanitize_key, TEKTON_HOME_RESULT_PATH, sanitize_key)
    return {
        "name": "copy-results-%s" % str(step_number),
        "args": args,
        "command": ["sh", "-c"],
        "image": TEKTON_COPY_RESULTS_STEP_IMAGE
    }


def _get_resourceOp_template(op: BaseOp,
                             name: str,
                             tekton_api_version: str,
                             resource_manifest: str,
                             owner_reference: str,
                             argo_var: bool = False):
    """Tekton task template for running resourceOp

    Return a Tekton task template for interacting Kubernetes resources.

    Args:
        op {BaseOp}: class that inherits from BaseOp
        name {str}: resourceOp name
        tekton_api_version {str}: Tekton API Version
        resource_manifest {str}: Kubernetes manifest file for deploying resources.
        argo_var {bool}: Check whether Argo variable replacement is necessary.

    Returns:
        Dict[Text, Any]
    """
    template = {
        'apiVersion': tekton_api_version,
        'kind': 'Task',
        'metadata': {'name': name},
        'spec': {
            "params": [
                {
                    "description": "Action on the resource",
                    "name": "action",
                    "type": "string"
                },
                {
                    "default": "strategic",
                    "description": "Merge strategy when using action patch",
                    "name": "merge-strategy",
                    "type": "string"
                },
                {
                    "default": "",
                    "description": "An express to retrieval data from resource.",
                    "name": "output",
                    "type": "string"
                },
                {
                    "default": "",
                    "description": "A label selector express to decide if the action on resource is success.",
                    "name": "success-condition",
                    "type": "string"
                },
                {
                    "default": "",
                    "description": "A label selector express to decide if the action on resource is failure.",
                    "name": "failure-condition",
                    "type": "string"
                },
                {
                    # This image is hosted by the kfp-tekton maintainers
                    # Source code: https://github.com/kubeflow/kfp-tekton/tree/master/tekton-catalog/kubectl-wrapper
                    "default": RESOURCE_OP_IMAGE,
                    "description": "Kubectl wrapper image",
                    "name": "image",
                    "type": "string"
                },
                {
                    "default": "false",
                    "description": "Enable set owner reference for created resource.",
                    "name": "set-ownerreference",
                    "type": "string"
                }
            ],
            'steps': [
                {
                    "command": [
                        "kubeclient"
                    ],
                    "args": [
                        "--action=$(params.action)",
                        "--merge-strategy=$(params.merge-strategy)",
                        "--manifest=%s" % resource_manifest,
                        "--output=$(params.output)",
                        "--success-condition=$(params.success-condition)",
                        "--failure-condition=$(params.failure-condition)",
                        "--set-ownerreference=$(params.set-ownerreference)"
                    ],
                    "image": "$(params.image)",
                    "name": "main",
                    "resources": {}
                }
            ]
        }
    }

    # Inject Argo variable replacement as env variables.
    if argo_var:
        template['spec']['steps'][0]['env'] = [
            {'name': 'PIPELINERUN', 'valueFrom': {'fieldRef': {'fieldPath': "metadata.labels['tekton.dev/pipelineRun']"}}}
        ]

    # Add results if exist.
    if op.attribute_outputs.items():
        template['spec']['results'] = []
        for output_item in sorted(list(op.attribute_outputs.items()), key=lambda x: x[0]):
            template['spec']['results'].append({'name': output_item[0], 'type': 'string', 'description': output_item[1]})

    if owner_reference == 'true':
        owner_reference_env = [
            {'name': 'POD_NAME', 'valueFrom': {'fieldRef': {'fieldPath': "metadata.name"}}},
            {'name': 'POD_NAMESPACE', 'valueFrom': {'fieldRef': {'fieldPath': "metadata.namespace"}}},
            {'name': 'POD_UID', 'valueFrom': {'fieldRef': {'fieldPath': "metadata.uid"}}}
        ]
        template['spec']['steps'][0]['env'] = template['spec']['steps'][0].get('env', [])
        template['spec']['steps'][0]['env'].extend(owner_reference_env)

    return template


def _add_mount_path(name: str,
                    path: str,
                    mount_path: str,
                    volume_mount_step_template: List[Dict[Text, Any]],
                    volume_template: List[Dict[Text, Any]],
                    mounted_param_paths: List[Text]):
    """
    Add emptyDir to the given mount_path for persisting files within the same tasks
    """
    volume_mount_step_template.append({'name': sanitize_k8s_name(name), 'mountPath': path.rsplit("/", 1)[0]})
    volume_template.append({'name': sanitize_k8s_name(name), 'emptyDir': {}})
    mounted_param_paths.append(mount_path)


def _update_volumes(template_spec: Dict[Text, Any],
                    volume_mount_step_template: List[Dict[Text, Any]],
                    volume_template: List[Dict[Text, Any]]):
    """
    Update the list of volumes and volumeMounts on a given template
    """
    if volume_mount_step_template:
        template_spec.setdefault('stepTemplate', {})
        template_spec['stepTemplate'].setdefault('volumeMounts', [])
        template_spec['stepTemplate']['volumeMounts'].extend(volume_mount_step_template)
        template_spec.setdefault('volumes', [])
        template_spec['volumes'].extend(volume_template)


def _prepend_steps(prep_steps: List[Dict[Text, Any]], original_steps: List[Dict[Text, Any]]):
    """
    Prepend steps to the original step list.
    """
    steps = prep_steps.copy()
    steps.extend(original_steps)
    return steps


def _process_parameters(processed_op: BaseOp,
                        template: Dict[Text, Any],
                        outputs_dict: Dict[Text, Any],
                        volume_mount_step_template: List[Dict[Text, Any]],
                        volume_template: List[Dict[Text, Any]],
                        replaced_param_list: List[Text],
                        artifact_to_result_mapping: Dict[Text, Any],
                        mounted_param_paths: List[Text],
                        tekton_bash_image_name: str = TEKTON_BASH_STEP_IMAGE):
    """Process output parameters to replicate the same behavior as Argo.

    Since Tekton results need to be under /tekton/results. If file output paths cannot be
    configured to /tekton/results, we need to create the below copy step for moving
    file outputs to the Tekton destination. BusyBox is recommended to be used on
    small tasks because it's relatively lightweight and small compared to the ubuntu and
    bash images.

    - image: busybox
        name: copy-results
        script: |
            #!/bin/sh
            set -exo pipefail
            cp $LOCALPATH $(results.data.path);

    Args:
        processed_op {BaseOp}: class that inherits from BaseOp
        template {Dict[Text, Any]}: Task template
        outputs_dict {Dict[Text, Any]}: Dictionary of the possible parameters/artifacts in this task
        volume_mount_step_template {List[Dict[Text, Any]]}: Step template for the list of volume mounts
        volume_template {List[Dict[Text, Any]]}: Task template for the list of volumes
        replaced_param_list {List[Text]}: List of parameters that already set up as results
        artifact_to_result_mapping {Dict[Text, Any]}: Mapping between parameter and artifact results
        mounted_param_paths {List[Text]}: List of paths that already mounted to a volume.

    Returns:
        Dict[Text, Any]
    """
    if outputs_dict.get('parameters'):
        template['spec']['results'] = []
        copy_results_step = _get_base_step('copy-results', tekton_bash_image_name)
        script = "set -exo pipefail\n"
        for name, path in processed_op.file_outputs.items():
            template['spec']['results'].append({
                'name': name,
                'type': 'string',
                'description': path
            })
            # replace all occurrences of the output file path with the Tekton output parameter expression
            need_copy_step = True
            for s in template['spec']['steps']:
                if 'command' in s:
                    commands = []
                    for c in s['command']:
                        if path in c:
                            c = c.replace(path, '$(results.%s.path)' % sanitize_k8s_name(name, allow_capital=True))
                            need_copy_step = False
                        commands.append(c)
                    s['command'] = commands
                if 'args' in s:
                    args = []
                    for a in s['args']:
                        if path in a:
                            a = a.replace(path, '$(results.%s.path)' % sanitize_k8s_name(name, allow_capital=True))
                            need_copy_step = False
                        args.append(a)
                    s['args'] = args
                if path == '/tekton/results/' + sanitize_k8s_name(name, allow_capital=True):
                    need_copy_step = False
            # If file output path cannot be found/replaced, use emptyDir to copy it to the tekton/results path
            if need_copy_step:
                script = script + 'cp ' + path + ' $(results.%s.path);\n' % sanitize_k8s_name(name, allow_capital=True)
                mount_path = path.rsplit("/", 1)[0]
                if mount_path not in mounted_param_paths:
                    _add_mount_path(name, path, mount_path, volume_mount_step_template, volume_template, mounted_param_paths)
            # Record what artifacts are moved to result parameters.
            parameter_name = sanitize_k8s_name(processed_op.name + '-' + name, allow_capital_underscore=True, max_length=float('Inf'))
            replaced_param_list.append(parameter_name)
            artifact_to_result_mapping[parameter_name] = name
        copy_results_step['command'].append(script)
        return copy_results_step
    else:
        return {}


def _process_output_artifacts(outputs_dict: Dict[Text, Any],
                              volume_mount_step_template: List[Dict[Text, Any]],
                              volume_template: List[Dict[Text, Any]],
                              replaced_param_list: List[Text],
                              artifact_to_result_mapping: Dict[Text, Any],
                              artifact_items: List[Any]):
    """Process output artifact dependencies to replicate the same behavior as Argo.

    For storing artifacts, we will need to provide the output artifact dependencies for the server to
    find and store the artifacts with the proper metadata.

    Args:
        outputs_dict {Dict[Text, Any]}: Dictionary of the possible parameters/artifacts in this task
        volume_mount_step_template {List[Dict[Text, Any]]}: Step template for the list of volume mounts
        volume_template {List[Dict[Text, Any]]}: Task template for the list of volumes
        replaced_param_list {List[Text]}: List of parameters that already set up as results
        artifact_to_result_mapping {Dict[Text, Any]}: Mapping between parameter and artifact results

    Returns:
        Dict[Text, Any]
    """

    if outputs_dict.get('artifacts'):
        mounted_artifact_paths = []
        for artifact in outputs_dict['artifacts']:
            parameter_name = sanitize_k8s_name(artifact['name'], allow_capital_underscore=True, max_length=float('Inf'))
            artifact_name = artifact_to_result_mapping.get(parameter_name, parameter_name)
            if parameter_name in replaced_param_list:
                artifact_items.append([artifact_name, "$(results.%s.path)" % sanitize_k8s_name(artifact_name, allow_capital=True)])
            else:
                artifact_items.append([artifact_name, artifact['path']])
                mount_path = artifact['path'].rsplit("/", 1)[0]
                if mount_path not in mounted_artifact_paths:
                    if mount_path == "":
                        raise ValueError('Undefined volume path or "/" path artifacts are not allowed.')
                    volume_mount_step_template.append({
                        'name': parameter_name, 'mountPath': mount_path
                    })
                    volume_template.append({'name': parameter_name, 'emptyDir': {}})
                    mounted_artifact_paths.append(mount_path)


def _process_base_ops(op: BaseOp):
    """Recursively go through the attrs listed in `attrs_with_pipelineparams`
    and sanitize and replace pipeline params with template var string.

    Returns a processed `BaseOp`.

    NOTE this is an in-place update to `BaseOp`'s attributes (i.e. the ones
    specified in `attrs_with_pipelineparams`, all `PipelineParam` are replaced
    with the corresponding template variable strings).

    Args:
        op {BaseOp}: class that inherits from BaseOp

    Returns:
        BaseOp
    """

    # map param's (unsanitized pattern or serialized str pattern) -> input param var str
    # Pick the shortest param full name if there's two identical params in the DSL
    map_to_tmpl_var = {}
    for param in op.inputs:
        key = (param.pattern or str(param))
        if not map_to_tmpl_var.get(key) or len('$(inputs.params.%s)' % param.full_name) < len(map_to_tmpl_var.get(key)):
            map_to_tmpl_var[key] = '$(inputs.params.%s)' % param.full_name

    # process all attr with pipelineParams except inputs and outputs parameters
    for key in op.attrs_with_pipelineparams:
        setattr(op, key, _process_obj(getattr(op, key), map_to_tmpl_var))

    return op


def _op_to_template(op: BaseOp,
                    pipelinerun_output_artifacts={},
                    artifact_items={},
                    generate_component_spec_annotations=True,
                    tekton_bash_image_name=TEKTON_BASH_STEP_IMAGE):
    """Generate template given an operator inherited from BaseOp."""

    # Display name
    if op.display_name:
        op.add_pod_annotation('pipelines.kubeflow.org/task_display_name', op.display_name)

    # initial local variables for tracking volumes and artifacts
    volume_mount_step_template = []
    volume_template = []
    mounted_param_paths = []
    replaced_param_list = []
    artifact_to_result_mapping = {}

    # NOTE in-place update to BaseOp
    # replace all PipelineParams with template var strings
    processed_op = _process_base_ops(op)

    if isinstance(op, dsl.ContainerOp):
        # default output artifacts
        output_artifact_paths = OrderedDict(op.output_artifact_paths)
        # print(op.output_artifact_paths)
        # This should have been as easy as output_artifact_paths.update(op.file_outputs),
        # but the _outputs_to_json function changes the output names and we must do the same here,
        # so that the names are the same
        output_artifact_paths.update(sorted(((param.full_name, processed_op.file_outputs[param.name])
                                             for param in processed_op.outputs.values()), key=lambda x: x[0]))

        output_artifacts = [
            {'name': name, 'path': path}
            for name, path in output_artifact_paths.items()
        ]

        # workflow template
        container = convert_k8s_obj_to_json(
            processed_op.container
        )

        # Calling containerOp step as "main" to align with Argo
        step = {'name': "main"}
        step.update(container)

        template = {
            'apiVersion': tekton_api_version,
            'kind': 'Task',
            'metadata': {'name': processed_op.name},
            'spec': {
                'steps': [step]
            }
        }

        # Create output artifact tracking annotation.
        for output_artifact in output_artifacts:
            output_annotation = pipelinerun_output_artifacts.get(processed_op.name, [])
            output_annotation.append(
                {
                    'name': output_artifact.get('name', ''),
                    'path': output_artifact.get('path', ''),
                    'key': "artifacts/$PIPELINERUN/%s/%s.tgz" %
                    (processed_op.name, output_artifact.get('name', '').replace(processed_op.name + '-', ''))
                }
            )
            pipelinerun_output_artifacts[processed_op.name] = output_annotation

    elif isinstance(op, dsl.ResourceOp):
        # no output artifacts
        output_artifacts = []

        # Flatten manifest because it needs to replace Argo variables
        manifest = yaml.dump(convert_k8s_obj_to_json(processed_op.k8s_resource), default_flow_style=False)
        argo_var = False
        if manifest.find('{{workflow.name}}') != -1:
            # Kubernetes Pod arguments only take $() as environment variables
            manifest = manifest.replace('{{workflow.name}}', "$(PIPELINERUN)")
            # Remove yaml quote in order to read bash variables
            manifest = re.sub('name: \'([^\']+)\'', 'name: \g<1>', manifest)
            argo_var = True

        # task template
        template = _get_resourceOp_template(op=op,
                                            name=processed_op.name,
                                            tekton_api_version=tekton_api_version,
                                            resource_manifest=manifest,
                                            owner_reference=op.resource.get('setOwnerReference'),
                                            argo_var=argo_var)

    # initContainers
    if processed_op.init_containers:
        template['spec']['steps'] = _prepend_steps(processed_op.init_containers, template['spec']['steps'])

    # inputs
    input_artifact_paths = processed_op.input_artifact_paths if isinstance(processed_op, dsl.ContainerOp) else None
    artifact_arguments = processed_op.artifact_arguments if isinstance(processed_op, dsl.ContainerOp) else None
    inputs = _inputs_to_json(processed_op.inputs, input_artifact_paths, artifact_arguments)
    if 'parameters' in inputs:
        if isinstance(processed_op, dsl.ContainerOp):
            template['spec']['params'] = inputs['parameters']
        elif isinstance(op, dsl.ResourceOp):
            template['spec']['params'].extend(inputs['parameters'])
    if 'artifacts' in inputs:
        # Leave artifacts for big data passing
        template['spec']['artifacts'] = inputs['artifacts']

    # outputs
    if isinstance(op, dsl.ContainerOp):
        op_outputs = processed_op.outputs
        param_outputs = processed_op.file_outputs
    elif isinstance(op, dsl.ResourceOp):
        op_outputs = {}
        param_outputs = {}
    outputs_dict = _outputs_to_json(op, op_outputs, param_outputs, output_artifacts)
    artifact_items[op.name] = artifact_items.get(op.name, [])
    if outputs_dict:
        copy_results_step = _process_parameters(processed_op,
                                                template,
                                                outputs_dict,
                                                volume_mount_step_template,
                                                volume_template,
                                                replaced_param_list,
                                                artifact_to_result_mapping,
                                                mounted_param_paths,
                                                tekton_bash_image_name)
        _process_output_artifacts(outputs_dict,
                                  volume_mount_step_template,
                                  volume_template,
                                  replaced_param_list,
                                  artifact_to_result_mapping,
                                  artifact_items[op.name])
        if mounted_param_paths:
            template['spec']['steps'].append(copy_results_step)
        _update_volumes(template['spec'], volume_mount_step_template, volume_template)

    # metadata
    if processed_op.pod_annotations or processed_op.pod_labels:
        template.setdefault('metadata', {})  # Tekton change, don't wipe out existing metadata
        if processed_op.pod_annotations:
            template['metadata']['annotations'] = {
                sanitize_k8s_name(key, allow_capital_underscore=True, allow_dot=True,
                                  allow_slash=True, max_length=253):
                    value
                for key, value in processed_op.pod_annotations.items()
            }
        if processed_op.pod_labels:
            template['metadata']['labels'] = {
                sanitize_k8s_name(key, allow_capital_underscore=True, allow_dot=True,
                                  allow_slash=True, max_length=253):
                    sanitize_k8s_name(value, allow_capital_underscore=True, allow_dot=True)
                for key, value in processed_op.pod_labels.items()
            }

    # sidecars
    if processed_op.sidecars:
        template['spec']['sidecars'] = processed_op.sidecars

    # volumes
    if processed_op.volumes:
        template['spec']['volumes'] = template['spec'].get('volumes', []) + [convert_k8s_obj_to_json(volume)
                                                                            for volume in processed_op.volumes]
        template['spec']['volumes'].sort(key=lambda x: x['name'])

    if isinstance(op, dsl.ContainerOp) and op._metadata and GENERATE_COMPONENT_SPEC_ANNOTATIONS and generate_component_spec_annotations:
        component_spec_dict = op._metadata.to_dict()
        component_spec_digest = hashlib.sha256(json.dumps(component_spec_dict, sort_keys=True).encode()).hexdigest()
        component_name = component_spec_dict.get('name', op.name)
        component_version = component_name + '@sha256=' + component_spec_digest
        digested_component_spec_dict = {'name': component_name,
                                        'outputs': component_spec_dict.get('outputs', []),
                                        'version': component_version
        }
        template.setdefault('metadata', {}).setdefault('annotations', {})['pipelines.kubeflow.org/component_spec_digest'] = \
            json.dumps(digested_component_spec_dict, sort_keys=True)
        if env.get('DISABLE_ARTIFACT_TRACKING', 'false').lower() == 'true':
            template['metadata']['annotations'].pop('pipelines.kubeflow.org/component_spec_digest', None)

    if isinstance(op, dsl.ContainerOp) and op.execution_options:
        if op.execution_options.caching_strategy.max_cache_staleness:
            template.setdefault('metadata', {}).setdefault('annotations', {})['pipelines.kubeflow.org/max_cache_staleness'] = \
                str(op.execution_options.caching_strategy.max_cache_staleness)

    # Sort and arrange results based on provided estimate size and process results in multi-steps if the result sizes are too big.
    result_size_map = "{}"
    if processed_op.pod_annotations:
        result_size_map = processed_op.pod_annotations.get("tekton-result-sizes", "{}")
    # Only sort and arrange results when the estimated sizes are given.
    if result_size_map and result_size_map != "{}":
        try:
            result_size_map = json.loads(result_size_map)
        except ValueError:
            raise ("tekton-result-sizes annotation is not a valid JSON")
        # Normalize estimated result size keys.
        result_size_map = {sanitize_k8s_name(key, allow_capital_underscore=True): value
                           for key, value in result_size_map.items()}
        # Sort key orders based on values
        result_size_map = dict(sorted(result_size_map.items(), key=lambda item: item[1], reverse=True))
        max_byte_size = 2048
        verified_result_size_map = {0: {}}
        op_result_names = [name['name'] for name in template['spec']['results']]
        step_bins = {0: 0}
        step_counter = 0
        # Group result files to not exceed max_byte_size as a bin packing problem
        # Results are sorted from large to small, each value will loop over each bin to determine can it fit in the existing bins.
        for key, value in result_size_map.items():
            try:
                value = int(value)
            except ValueError:
                raise ("Estimated value for result %s is %s, but it needs to be an integer." % (key, value))
            if key in op_result_names:
                packed_index = -1
                # Look for bin that can fit the result value
                for i in range(len(step_bins)):
                    if step_bins[i] + value > max_byte_size:
                        continue
                    step_bins[i] = step_bins[i] + value
                    packed_index = i
                    break
                # If no bin can fit the value, create a new bin to store the value
                if packed_index < 0:
                    step_counter += 1
                    if value > max_byte_size:
                        logging.warning("The estimated size for parameter %s is %sB which is more than 2KB, "
                            "consider passing this value as artifact instead of output parameter." % (key, str(value)))
                    step_bins[step_counter] = value
                    verified_result_size_map[step_counter] = {}
                    packed_index = step_counter
                verified_result_size_map[packed_index][key] = value
            else:
                logging.warning("The esitmated size for parameter %s does not exist in the task %s."
                            "Please correct the task annotations with the correct parameter key" % (key, op.name))
        missing_param_estimation = []
        for result_name in op_result_names:
            if result_name not in result_size_map.keys():
                missing_param_estimation.append(result_name)
        if missing_param_estimation:
            logging.warning("The following output parameter estimations are missing in task %s: Missing params: %s."
                         % (op.name, missing_param_estimation))
        # Move results between the Tekton home and result directories if there are more than one step
        if step_counter > 0:
            for step in template['spec']['steps']:
                if step['name'] == 'main':
                    for key in result_size_map.keys():
                        # Replace main step results that are not in the first bin to the Tekton home path
                        if key not in verified_result_size_map[0].keys():
                            sanitize_key = sanitize_k8s_name(key, allow_capital=True)
                            for i, a in enumerate(step['args']):
                                a = a.replace('$(results.%s.path)' % sanitize_key, '%s%s' % (TEKTON_HOME_RESULT_PATH, sanitize_key))
                                step['args'][i] = a
                            for i, c in enumerate(step['command']):
                                c = c.replace('$(results.%s.path)' % sanitize_key, '%s%s' % (TEKTON_HOME_RESULT_PATH, sanitize_key))
                                step['command'][i] = c
            # Append new steps to move result files between each step, so Tekton controller can record all results without
            # exceeding the Kubernetes termination log limit.
            for i in range(1, step_counter + 1):
                copy_result_step = _get_copy_result_step_template(i, verified_result_size_map)
                template['spec']['steps'].append(copy_result_step)
        # Update actifact item location to the latest stage in order to properly track and store all the artifacts.
        for i, artifact in enumerate(artifact_items[op.name]):
            if artifact[0] not in verified_result_size_map[step_counter].keys():
                artifact[1] = '%s%s' % (TEKTON_HOME_RESULT_PATH, sanitize_k8s_name(artifact[0], allow_capital=True))
                artifact_items[op.name][i] = artifact
    # Add workspaces into task spec if defined as annotations
    workspace_map = "{}"
    if processed_op.pod_annotations:
        workspace_map = processed_op.pod_annotations.get("workspaces", "{}")
    if workspace_map and workspace_map != "{}":
        try:
            workspace_map = json.loads(workspace_map)
        except ValueError:
            raise ("workspace_map annotation is not a valid JSON")
        workspaces = template['spec'].get("workspaces", [])
        for key, value in workspace_map.items():
            workspace_item = {"name": key}
            if value.get("readOnly", None) is not None:
                workspace_item["readOnly"] = value["readOnly"]
            if value.get("mountPath", None) is not None:
                workspace_item["mountPath"] = value["mountPath"]
            workspaces.append(workspace_item)
        template['spec']["workspaces"] = workspaces
        processed_op.pod_annotations.pop("workspaces")
        template['metadata']['annotations'].pop("workspaces")
    return template
