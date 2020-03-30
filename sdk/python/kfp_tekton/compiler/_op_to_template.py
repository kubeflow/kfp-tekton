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
from collections import OrderedDict

from kfp.compiler._k8s_helper import convert_k8s_obj_to_json
from kfp.compiler._op_to_template import _process_obj, _inputs_to_json, _outputs_to_json
from kfp import dsl
from kfp.dsl import ArtifactLocation
from kfp.dsl._container_op import BaseOp

from .. import tekton_api_version


class literal_str(str):
    """
    Literal dumper for pyyaml
    """
    pass


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
    map_to_tmpl_var = {
        (param.pattern or str(param)): '$(inputs.params.%s)' % param.full_name  # Tekton change
        for param in op.inputs
    }

    # process all attr with pipelineParams except inputs and outputs parameters
    for key in op.attrs_with_pipelineparams:
        setattr(op, key, _process_obj(getattr(op, key), map_to_tmpl_var))

    return op


def _op_to_template(op: BaseOp):
    """Generate template given an operator inherited from BaseOp."""

    # NOTE in-place update to BaseOp
    # replace all PipelineParams with template var strings
    processed_op = _process_base_ops(op)

    if isinstance(op, dsl.ContainerOp):
        # default output artifacts
        output_artifact_paths = OrderedDict(op.output_artifact_paths)
        # print(op.output_artifact_paths)
        # This should have been as easy as output_artifact_paths.update(op.file_outputs), but the _outputs_to_json function changes the output names and we must do the same here, so that the names are the same
        output_artifact_paths.update(sorted(((param.full_name, processed_op.file_outputs[param.name]) for param in processed_op.outputs.values()), key=lambda x: x[0]))

        output_artifacts = [
            #  convert_k8s_obj_to_json(
            #      ArtifactLocation.create_artifact_for_s3(
            #          op.artifact_location,
            #          name=name,
            #          path=path,
            #          key='runs/{{workflow.uid}}/{{pod.name}}/' + name + '.tgz'))
            # for name, path in output_artifact_paths.items()
        ]

        # workflow template
        container = convert_k8s_obj_to_json(
            processed_op.container
        )

        step = {'name': processed_op.name}
        step.update(container)

        template = {
            'apiVersion': tekton_api_version,
            'kind': 'Task',
            'metadata': {'name': processed_op.name},
            'spec': {
                'steps': [step]
            }
        }

    elif isinstance(op, dsl.ResourceOp):
        # # no output artifacts
        # output_artifacts = []
        #
        # # workflow template
        # processed_op.resource["manifest"] = yaml.dump(
        #     convert_k8s_obj_to_json(processed_op.k8s_resource),
        #     default_flow_style=False
        # )
        # template = {
        #     'name': processed_op.name,
        #     'resource': convert_k8s_obj_to_json(
        #         processed_op.resource
        #     )
        # }
        raise NotImplementedError("dsl.ResourceOp is not yet implemented")

    # initContainers
    if processed_op.init_containers:
        steps = processed_op.init_containers.copy()
        steps.extend(template['spec']['steps'])
        template['spec']['steps'] = steps

    # inputs
    input_artifact_paths = processed_op.input_artifact_paths if isinstance(processed_op, dsl.ContainerOp) else None
    artifact_arguments = processed_op.artifact_arguments if isinstance(processed_op, dsl.ContainerOp) else None
    inputs = _inputs_to_json(processed_op.inputs, input_artifact_paths, artifact_arguments)
    if 'parameters' in inputs:
        template['spec']['params'] = inputs['parameters']
    elif 'artifacts' in inputs:
        raise NotImplementedError("input artifacts are not yet implemented")

    # outputs
    if isinstance(op, dsl.ContainerOp):
        param_outputs = processed_op.file_outputs
    elif isinstance(op, dsl.ResourceOp):
        param_outputs = processed_op.attribute_outputs
    outputs_dict = _outputs_to_json(op, processed_op.outputs, param_outputs, output_artifacts)
    if outputs_dict:
        """
        Since Tekton results need to be under /tekton/results. If file output paths cannot be
        configured to /tekton/results, we need to create the below copy step for moving
        file outputs to the Tekton destination.

        - image: busybox
          name: copy-results
          script: |
            #!/bin/sh
            set -exo pipefail
            cp $LOCALPATH $(results.data.path);
        """
        template['spec']['results'] = []
        copy_results_step = {
            'image': 'busybox',
            'name': 'copy-results',
            'script': '#!/bin/sh\nset -exo pipefail\n'
        }
        volume_mount_step_template = []
        volume_template = []
        mounted_paths = []
        for name, path in processed_op.file_outputs.items():
            name = name.replace('_', '-')  # replace '_' to '-' since tekton results doesn't support underscore
            template['spec']['results'].append({
                'name': name,
                'description': path
            })
            # replace all occurrences of the output file path with the Tekton output parameter expression
            need_copy_step = True
            for s in template['spec']['steps']:
                if 'command' in s:
                    commands = []
                    for c in s['command']:
                        if path in c:
                            c = c.replace(path, '$(results.%s.path)' % name)
                            need_copy_step = False
                        commands.append(c)
                    s['command'] = commands
                if 'args' in s:
                    args = []
                    for a in s['args']:
                        if path in a:
                            a = a.replace(path, '$(results.%s.path)' % name)
                            need_copy_step = False
                        args.append(a)
                    s['args'] = args
            # If file output path cannot be configure, use emptyDir to copy it to the tekton/results path.
            if need_copy_step:
                copy_results_step['script'] = copy_results_step['script'] + 'cp ' + path + ' $(results.%s.path);' % name + '\n'
                mountPath = path.rsplit("/", 1)[0]
                if mountPath not in mounted_paths:
                    volume_mount_step_template.append({'name': name, 'mountPath': path.rsplit("/", 1)[0]})
                    volume_template.append({'name': name, 'emptyDir': {}})
                    mounted_paths.append(mountPath)
        if mounted_paths:
            copy_results_step['script'] = literal_str(copy_results_step['script'])
            template['spec']['steps'].append(copy_results_step)
            template['spec']['stepTemplate'] = {}
            template['spec']['stepTemplate']['volumeMounts'] = volume_mount_step_template
            template['spec']['volumes'] = volume_template

    # **********************************************************
    #  NOTE: the following features are still under development
    # **********************************************************

    # node selector
    if processed_op.node_selector:
        raise NotImplementedError("'nodeSelector' is not (yet) implemented")
        template['nodeSelector'] = processed_op.node_selector

    # tolerations
    if processed_op.tolerations:
        raise NotImplementedError("'tolerations' is not (yet) implemented")
        template['tolerations'] = processed_op.tolerations

    # affinity
    if processed_op.affinity:
        raise NotImplementedError("'affinity' is not (yet) implemented")
        template['affinity'] = convert_k8s_obj_to_json(processed_op.affinity)

    # metadata
    if processed_op.pod_annotations or processed_op.pod_labels:
        template.setdefault('metadata', {})  # Tekton change, don't wipe out existing metadata
        if processed_op.pod_annotations:
            template['metadata']['annotations'] = processed_op.pod_annotations
        if processed_op.pod_labels:
            template['metadata']['labels'] = processed_op.pod_labels

    # sidecars
    if processed_op.sidecars:
        template['spec']['sidecars'] = processed_op.sidecars

    # volumes
    if processed_op.volumes:
        template['spec']['volumes'] = template['spec'].get('volume', []) + [convert_k8s_obj_to_json(volume) for volume in processed_op.volumes]
        template['spec']['volumes'].sort(key=lambda x: x['name'])

    # Display name
    if processed_op.display_name:
        template.setdefault('metadata', {}).setdefault('annotations', {})['pipelines.kubeflow.org/task_display_name'] = processed_op.display_name

    if isinstance(op, dsl.ContainerOp) and op._metadata:
        import json
        template.setdefault('metadata', {}).setdefault('annotations', {})['pipelines.kubeflow.org/component_spec'] = json.dumps(op._metadata.to_dict(), sort_keys=True)

    return template
