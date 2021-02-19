# Copyright 2020 The Kubeflow Authors.
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

from kfp import components
import kfp.dsl as dsl

from kubeflow.katib import ApiClient
from kubeflow.katib import V1beta1ExperimentSpec
from kubeflow.katib import V1beta1AlgorithmSpec
from kubeflow.katib import V1beta1ObjectiveSpec
from kubeflow.katib import V1beta1ParameterSpec
from kubeflow.katib import V1beta1FeasibleSpace
from kubeflow.katib import V1beta1TrialTemplate
from kubeflow.katib import V1beta1TrialParameterSpec


def generate_trial_template():
    # JSON template specification for the Trial's Worker Kubernetes Job.
    trial_spec = {
        "apiVersion": "batch/v1",
        "kind": "Job",
        "spec": {
            "template": {
                "metadata": {
                    "annotations": {
                        "sidecar.istio.io/inject": "false"
                    }
                },
                "spec": {
                    "containers": [
                        {
                            "name": "training-container",
                            "image": "docker.io/kubeflowkatib/mxnet-mnist:v1beta1-e294a90",
                            "command": [
                                "python3",
                                "/opt/mxnet-mnist/mnist.py",
                                "--batch-size=64",
                                "--lr=${trialParameters.learningRate}",
                                "--num-layers=${trialParameters.numberLayers}",
                                "--optimizer=${trialParameters.optimizer}"
                            ]
                        }
                    ],
                    "restartPolicy": "Never"
                }
            }
        }
    }

    # Configure parameters for the Trial template.
    # We can set the retain parameter to "True" to not clean-up the Trial Job's Kubernetes Pods.
    trial_template = V1beta1TrialTemplate(
        # retain=True,
        primary_container_name="training-container",
        success_condition='status.conditions.#(type=="Succeeded")#|#(status=="True")#',
        failure_condition='status.conditions.#(type=="Failed")#|#(status=="True")#',
        trial_parameters=[
            V1beta1TrialParameterSpec(
                name="learningRate",
                description="Learning rate for the training model",
                reference="lr"
            ),
            V1beta1TrialParameterSpec(
                name="numberLayers",
                description="Number of training model layers",
                reference="num-layers"
            ),
            V1beta1TrialParameterSpec(
                name="optimizer",
                description="Training model optimizer (sdg, adam or ftrl)",
                reference="optimizer"
            ),
        ],
        trial_spec=trial_spec
    )
    return trial_template


def experiment_search_params():
    # Experiment search space.
    # In this example we tune learning rate, number of layers and optimizer(sgd/adam/ftrl).
    return [
        V1beta1ParameterSpec(
            name="lr",
            parameter_type="double",
            feasible_space=V1beta1FeasibleSpace(
                min="0.01",
                max="0.03"
            ),
        ),
        V1beta1ParameterSpec(
            name="num-layers",
            parameter_type="int",
            feasible_space=V1beta1FeasibleSpace(
                min="2",
                max="5"
            ),
        ),
        V1beta1ParameterSpec(
            name="optimizer",
            parameter_type="categorical",
            feasible_space=V1beta1FeasibleSpace(
                list=['sgd', 'adam', 'ftrl']
            )
        )
    ]


def algorithm_spec():
    # Algorithm specification.
    return V1beta1AlgorithmSpec(
        algorithm_name="random"
    )


def objective_spec(goal):
    # Objective specification.
    return V1beta1ObjectiveSpec(
        type="maximize",
        goal=goal,
        objective_metric_name="Validation-accuracy",
        additional_metric_names=["accuracy"]
    )


@dsl.pipeline(
    name="Launch katib experiment",
    description="An example to launch katib experiment."
)
def mnist_hpo(
        name="mnist",
        namespace="anonymous",
        goal=0.99,
        parallel_trial_count=3,
        max_trial_count=12,
        experiment_timeout_minutes=60,
        delete_after_done=True):

    experiment_spec = V1beta1ExperimentSpec(
        max_trial_count=max_trial_count,
        max_failed_trial_count=3,
        parallel_trial_count=parallel_trial_count,
        objective=objective_spec(goal),
        algorithm=algorithm_spec(),
        parameters=experiment_search_params(),
        trial_template=generate_trial_template()
    )

    # Get the Katib launcher.
    katib_experiment_launcher_op = components.load_component_from_url(
        "https://raw.githubusercontent.com/kubeflow/pipelines/master/components/kubeflow/katib-launcher/component.yaml")

    # Experiment Spec should be serialized to a valid Kubernetes object.
    op = katib_experiment_launcher_op(
        experiment_name=name,
        experiment_namespace=namespace,
        experiment_spec=ApiClient().sanitize_for_serialization(experiment_spec),
        experiment_timeout_minutes=experiment_timeout_minutes,
        delete_finished_experiment=delete_after_done)

    # Output container to print the results.
    dsl.ContainerOp(
        name="best-hp",
        image="library/bash:4.4.23",
        command=["sh", "-c"],
        arguments=["echo Best HyperParameters: %s" % op.output],
    )


if __name__ == '__main__':
    from kfp_tekton.compiler import TektonCompiler
    TektonCompiler().compile(mnist_hpo, __file__.replace('.py', '.yaml'))
