# KFP Tekton Samples

Below are the list of samples that are currently running end to end taking the compiled Tekton yaml and deploying on a Tekton cluster directly.
If you are interested more in the larger list of pipelines samples we are testing for whether they can be 'compiled to Tekton' format, please [look at the corresponding status page](/sdk/python/tests/README.md)

[KFP Tekton User Guide](/guides/kfp-user-guide) is a guideline for the possible ways to develop and consume Kubeflow Pipeline with Tekton. It's recommended to go over at least one of the methods in the user guide before heading into the KFP Tekton Samples.

## Prerequisites
- Install [Kubeflow 1.3+ with KFP Tekton backend](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow-on-iks/#installation) or install [standalone kfp-tekton 0.7.0+](/guides/kfp_tekton_install.md#standalone-kubeflow-pipelines-with-tekton-backend-deployment). Then connect the cluster to the current shell with `kubectl`
- Install [kfp-tekton](/sdk/README.md) SDK
    ```
    # Set up the python virtual environment
    python3 -m venv .venv
    source .venv/bin/activate

    # Install the kfp-tekton SDK
    pip install kfp-tekton
    ```

## Samples

+ [MNIST End to End example with Kubeflow components](/samples/e2e-mnist)
+ [Hyperparameter tuning using Katib](/samples/katib)
+ [Trusted AI Pipeline with AI Fairness 360 and Adversarial Robustness 360 components](/samples/trusted-ai)
+ [Training and Serving Models with Watson Machine Learning](/samples/watson-train-serve#training-and-serving-models-with-watson-machine-learning)
+ [Lightweight python components example](/samples/lightweight-component)
+ [The flip-coin pipeline](/samples/flip-coin)
+ [Nested pipeline example](/samples/nested-pipeline)
+ [Pipeline with Nested loops](/samples/nested-loops)
