# TODO: from KFP 1.3.0, need to implement for kfp_tekton.compiler

# Copyright 2020 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from pathlib import Path

import kfp as kfp
from kfp.components import load_component_from_file, create_component_from_func
from typing import NamedTuple

test_data_dir = Path(__file__).parent / 'test_data'
producer_op = load_component_from_file(
    str(test_data_dir / 'produce_2.component.yaml'))
processor_op = load_component_from_file(
    str(test_data_dir / 'process_2_2.component.yaml'))
consumer_op = load_component_from_file(
    str(test_data_dir / 'consume_2.component.yaml'))


def metadata_and_metrics() -> NamedTuple(
    "Outputs",
    [("mlpipeline_ui_metadata", "UI_metadata"), ("mlpipeline_metrics", "Metrics"
                                                )],
):
    metadata = {
        "outputs": [{
            "storage": "inline",
            "source": "*this should be bold*",
            "type": "markdown"
        }]
    }
    metrics = {
        "metrics": [
            {
                "name": "train-accuracy",
                "numberValue": 0.9,
            },
            {
                "name": "test-accuracy",
                "numberValue": 0.7,
            },
        ]
    }
    from collections import namedtuple
    import json

    return namedtuple("output",
                      ["mlpipeline_ui_metadata", "mlpipeline_metrics"])(
                          json.dumps(metadata), json.dumps(metrics))


@kfp.dsl.pipeline()
def artifact_passing_pipeline():
    producer_task = producer_op()
    processor_task = processor_op(producer_task.outputs['output_1'],
                                  producer_task.outputs['output_2'])
    consumer_task = consumer_op(processor_task.outputs['output_1'],
                                processor_task.outputs['output_2'])

    markdown_task = create_component_from_func(func=metadata_and_metrics)()
    # This line is only needed for compiling using dsl-compile to work
    kfp.dsl.get_pipeline_conf(
    ).data_passing_method = volume_based_data_passing_method


from kubernetes.client.models import V1Volume, V1PersistentVolumeClaimVolumeSource
from kfp.dsl import data_passing_methods

volume_based_data_passing_method = data_passing_methods.KubernetesVolume(
    volume=V1Volume(
        name='data',
        persistent_volume_claim=V1PersistentVolumeClaimVolumeSource(
            claim_name='data-volume',),
    ),
    path_prefix='artifact_data/',
)


if __name__ == '__main__':
    pipeline_conf = kfp.dsl.PipelineConf()
    pipeline_conf.data_passing_method = volume_based_data_passing_method
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(artifact_passing_pipeline, __file__.replace('.py', '.yaml'), pipeline_conf=pipeline_conf)
