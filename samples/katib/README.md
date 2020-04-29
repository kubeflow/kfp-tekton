# Hyperparameter tuning using Katib

Katib is a Kubernetes-based system for Hyperparameter Tuning and Neural Architecture Search. This pipeline demonstrates how to use the Katib component to find the hyperparameter tuning results. 

This pipeline uses the [MNIST dataset](http://yann.lecun.com/exdb/mnist/) to train the model and try to find the best hyperparameters using random search. Learning rate, number of convolutional layers, and the optimizer are the training parameters we want to search. Once the pipeline is completed, the ending step will print out the best hyperparameters in this pipeline experiment and clean up the workspace.

## Prerequisites 
- Install [Kubeflow 1.0.2+](https://www.kubeflow.org/docs/started/getting-started/) and connect the cluster to the current shell with `kubectl`
- Install [Tekton 0.11.3](https://github.com/tektoncd/pipeline/releases/tag/v0.11.3) and [Tekton CLI](https://github.com/tektoncd/cli)
- Install [kfp-tekton](/sdk/README.md#steps) SDK

## Instructions

1. First, go to the Kubeflow dashboard and create a user namespace. The Kubeflow dashboard is the endpoint to your istio-ingressgateway. We will be using the namespace `anonymous` for this example.

2. Compile the Katib pipeline
```shell
dsl-compile-tekton --py katib.py --output katib.yaml
```

3. Run the Katib pipeline, click the `enter` key to use the default pipeline variables.
```shell
tkn pipeline start launch-katib-experiment -s default-editor -n anonymous --showlog
```

This pipeline will run for 10 to 15 minutes, then you should able to see the best hyperparameter tuning result at the end of the logs.
```
$ tkn pipeline start launch-katib-experiment -s default-editor -n anonymous --showlog
? Value for param `name` of type `string`? (Default is `mnist`) mnist
? Value for param `namespace` of type `string`? (Default is `anonymous`) anonymous
? Value for param `goal` of type `string`? (Default is `0.99`) 0.99
? Value for param `parallelTrialCount` of type `string`? (Default is `3`) 3
? Value for param `maxTrialCount` of type `string`? (Default is `12`) 12
? Value for param `experimentTimeoutMinutes` of type `string`? (Default is `60`) 60
? Value for param `deleteAfterDone` of type `string`? (Default is `True`) True
Pipelinerun started: launch-katib-experiment-run-8h9mc
Waiting for logs to be available...
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Generating experiment template.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Creating kubeflow.org/experiments mnist in namespace anonymous.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Created kubeflow.org/experiments mnist in namespace anonymous.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Created.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments mnist in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:kubeflow.org/experiments mnist in namespace anonymous has reached the expected condition: Succeeded.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Deleteing kubeflow.org/experiments mnist in namespace anonymous.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Deleted kubeflow.org/experiments mnist in namespace anonymous.

[my-out-cop : my-out-cop] hyperparameter: [{name: --lr, value: 0.02509986980552496}, {name: --num-layers, value: 3}, {name: --optimizer, value: sgd}]
```

## Acknowledgements

Thanks [Hougang Liu](https://github.com/hougangliu) for creating the original katib example.
