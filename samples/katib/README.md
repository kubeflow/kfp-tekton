# Hyperparameter tuning using Katib

[Katib](https://github.com/kubeflow/katib) is a Kubernetes-based system for Hyperparameter Tuning and Neural Architecture Search. This pipeline demonstrates how to use the Katib component to find the hyperparameter tuning results. 

This pipeline uses the [MNIST dataset](http://yann.lecun.com/exdb/mnist/) to train the model and try to find the best hyperparameters using random search. Learning rate, number of convolutional layers, and the optimizer are the training parameters we want to search. Once the pipeline is completed, the ending step will print out the best hyperparameters in this pipeline experiment and clean up the workspace.

## Prerequisites 
- Install [Kubeflow 1.0.2+](https://www.kubeflow.org/docs/started/getting-started/) and connect the cluster to the current shell with `kubectl`
- Install [Tekton 0.13.0](https://github.com/tektoncd/pipeline/releases/tag/v0.13.0)
- Install [kfp-tekton](/sdk/README.md#steps) SDK and [Kubeflow pipeline with Tekton backend](/tekton_kfp_guide.md)

## Instructions

1. First, go to the Kubeflow dashboard and create a user namespace. The Kubeflow dashboard is the endpoint to your istio-ingressgateway. We will be using the namespace `anonymous` for this example.

2. Compile the Katib pipeline
```shell
dsl-compile-tekton --py katib.py --output katib.yaml
```

3. Next, upload the `katib.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline. This pipeline will run for 10 to 15 minutes.

## Acknowledgements

Thanks [Hougang Liu](https://github.com/hougangliu) for creating the original katib example.
