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

import json
import kfp.dsl as dsl


@dsl.pipeline(
    name="Launch katib experiment",
    description="An example to launch katib experiment."
)
def mnist_hpo(
        name="mnist",
        namespace="kubeflow",
        goal: float = 0.99,
        parallelTrialCount: int = 3,
        maxTrialCount: int = 12,
        experimentTimeoutMinutes: int = 60,
        deleteAfterDone: bool = True):
    objectiveConfig = {
      "type": "maximize",
      "goal": goal,
      "objectiveMetricName": "Validation-accuracy",
      "additionalMetricNames": ["accuracy"]
    }
    algorithmConfig = {"algorithmName": "random"}
    parameters = [
      {"name": "--lr", "parameterType": "double", "feasibleSpace": {"min": "0.01", "max": "0.03"}},
      {"name": "--num-layers", "parameterType": "int", "feasibleSpace": {"min": "2", "max": "5"}},
      {"name": "--optimizer", "parameterType": "categorical", "feasibleSpace": {"list": ["sgd", "adam", "ftrl"]}}
    ]
    rawTemplate = {
      "apiVersion": "batch/v1",
      "kind": "Job",
      "metadata": {
         "name": "{{.Trial}}",
         "namespace": "{{.NameSpace}}"
      },
      "spec": {
        "template": {
          "spec": {
            "restartPolicy": "Never",
            "containers": [
              {"name": "{{.Trial}}",
               "image": "docker.io/katib/mxnet-mnist-example",
               "command": [
                   "python /mxnet/example/image-classification/train_mnist.py --batch-size=64 {{- with .HyperParameters}} {{- range .}} {{.Name}}={{.Value}} {{- end}} {{- end}}"  # noqa E501
               ]
              }
            ]
          }
        }
      }
    }
    trialTemplate = {
      "goTemplate": {
        "rawTemplate": json.dumps(rawTemplate)
      }
    }
    op1 = katib_experiment_launcher_op(
            name,
            namespace,
            parallelTrialCount=parallelTrialCount,
            maxTrialCount=maxTrialCount,
            objectiveConfig=str(objectiveConfig),
            algorithmConfig=str(algorithmConfig),
            trialTemplate=str(trialTemplate),
            parameters=str(parameters),
            experimentTimeoutMinutes=experimentTimeoutMinutes,
            deleteAfterDone=deleteAfterDone
    )

    op_out = dsl.ContainerOp(
        name="my-out-cop",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo hyperparameter: %s" % op1.output],
    )


def katib_experiment_launcher_op(
      name,
      namespace,
      maxTrialCount=100,
      parallelTrialCount=3,
      maxFailedTrialCount=3,
      objectiveConfig='{}',
      algorithmConfig='{}',
      metricsCollector='{}',
      trialTemplate='{}',
      parameters='[]',
      experimentTimeoutMinutes=60,
      deleteAfterDone=True,
      outputFile='/output.txt'):
    return dsl.ContainerOp(
        name="mnist-hpo",
        image='liuhougangxa/katib-experiment-launcher:latest',
        arguments=[
            '--name', name,
            '--namespace', namespace,
            '--maxTrialCount', maxTrialCount,
            '--maxFailedTrialCount', maxFailedTrialCount,
            '--parallelTrialCount', parallelTrialCount,
            '--objectiveConfig', objectiveConfig,
            '--algorithmConfig', algorithmConfig,
            '--metricsCollector', metricsCollector,
            '--trialTemplate', trialTemplate,
            '--parameters', parameters,
            '--outputFile', outputFile,
            '--deleteAfterDone', deleteAfterDone,
            '--experimentTimeoutMinutes', experimentTimeoutMinutes,
        ],
        file_outputs={'bestHyperParameter': outputFile}
    )


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(mnist_hpo, __file__.replace('.py', '.yaml'))
