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
from kfp import dsl, components


@dsl.pipeline(
    name="launch-katib-experiment",
    description="An example to launch katib experiment."
)
def mnist_hpo(
        name: str = "mnist",
        namespace: str = "kubeflow",
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

    op_out = components.load_component_from_text("""
      name: my-out-cop
      description: output component
      inputs:
        - {name: text, type: String}
      implementation:
        container:
          image: library/bash:4.4.23
          command:
          - sh
          - -c
          - |
            echo hyperparameter: $0
          - {inputValue: text}
      """)(op1.outputs['bestHyperParameter'])


def katib_experiment_launcher_op(
      name: str,
      namespace: str,
      maxTrialCount: int = 100,
      parallelTrialCount: int = 3,
      maxFailedTrialCount: int = 3,
      objectiveConfig: str = '{}',
      algorithmConfig: str = '{}',
      metricsCollector: str = '{}',
      trialTemplate: str = '{}',
      parameters: str = '[]',
      experimentTimeoutMinutes: int = 60,
      deleteAfterDone: bool = True):

    component_str = """
    name: mnist-hpo
    description: mnist hpo
    inputs:
      - {name: name, type: String}
      - {name: namespace, type: String}
      - {name: maxtrialcount, type: Integer}
      - {name: maxfailedtrialcount, type: Integer}
      - {name: paralleltrialcount, type: Integer}
      - {name: objectiveconfig, type: String}
      - {name: algorithmconfig, type: String}
      - {name: metricscollector, type: String}
      - {name: trialtemplate, type: String}
      - {name: parameters, type: String}
      - {name: deleteafterdone, type: Boolean}
      - {name: experimenttimeoutminutes, type: Integer}
    outputs:
      - {name: bestHyperParameter, type: String}
    implementation:
      container:
        image: liuhougangxa/katib-experiment-launcher:latest
        args:
        - --name
        - {inputValue: name}
        - --namespace
        - {inputValue: namespace}
        - --maxTrialCount
        - {inputValue: maxtrialcount}
        - --maxFailedTrialCount
        - {inputValue: maxfailedtrialcount}
        - --parallelTrialCount
        - {inputValue: paralleltrialcount}
        - --objectiveConfig
        - {inputValue: objectiveconfig}
        - --algorithmConfig
        - {inputValue: algorithmconfig}
        - --metricsCollector
        - {inputValue: metricscollector}
        - --trialTemplate
        - {inputValue: trialtemplate}
        - --parameters
        - {inputValue: parameters}
        - --outputFile
        - {outputPath: bestHyperParameter}
        - --deleteAfterDone
        - {inputValue: deleteafterdone}
        - --experimentTimeoutMinutes
        - {inputValue: experimenttimeoutminutes}
    """
    return components.load_component_from_text(component_str)(
      name=name,
      namespace=namespace,
      maxtrialcount=maxTrialCount,
      maxfailedtrialcount=maxFailedTrialCount,
      paralleltrialcount=parallelTrialCount,
      objectiveconfig=objectiveConfig,
      algorithmconfig=algorithmConfig,
      metricscollector=metricsCollector,
      trialtemplate=trialTemplate,
      parameters=parameters,
      deleteafterdone=deleteAfterDone,
      experimenttimeoutminutes=experimentTimeoutMinutes)


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(mnist_hpo, __file__.replace('.py', '.yaml'))
