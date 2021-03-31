# Trusted AI pipeline with AI Fairness 360 and Adversarial Robustness 360 components

Artificial intelligence is becoming a crucial component of enterprises’ operations and strategy. In order to responsibly take advantage of AI today, we must figure out ways to instill transparency, explainability, fairness, and robustness into AI. In this example, we will be going over how to produce fairness and robustness metrics using the Trusted AI libraries from [AI Fairness 360](https://github.com/IBM/AIF360) and [Adversarial Robustness Toolbox](https://github.com/IBM/adversarial-robustness-toolbox).

This pipeline uses the [UTKface's aligned & cropped faces dataset](https://susanqq.github.io/UTKFace/) to train a gender classification model using the Katib engine. Once the training is completed, there will be two extra tasks that use the stored model and dataset to produce fairness and robustness metrics.

## Prerequisites 
- Install [KFP Tekton prerequisites](/samples/README.md)
- If you are using the single-user deployment of Kubeflow or the standalone deployment of KFP-Tekton then create this cluster role binding.
```shell
kubectl create clusterrolebinding pipeline-runner-extend --clusterrole cluster-admin --serviceaccount=kubeflow:pipeline-runner
```

## Instructions

1. First, go to the Kubeflow dashboard and create a user namespace. The Kubeflow dashboard is the endpoint to your istio-ingressgateway. We will be using the namespace `anonymous` for this example.

2. Compile the trusted-ai pipeline. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `trusted-ai.yaml`.
```shell
python trusted-ai.py
```

3. Next, upload the `trusted-ai.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.

Below are the metrics definition for this example:
**Fairness Metrics**
- **Classification accuracy**: Amount of correct predictions using the test data. Ideal value: 1
- **Balanced classification accuracy**: Balanced true positive and negative predictions (0.5*(TPR+TNR)) using the test data. Ideal value: 1
- **Statistical parity difference**: Difference of the rate of favorable outcomes received by the unprivileged group to the privileged group. Ideal value: 0 (-0.1 to 0.1 will consider as fair)
- **Disparate impact**: The ratio of rate of favorable outcome for the unprivileged group to that of the privileged group. Ideal value: 1 (0.8 to 1.2 will consider as fair)
- **Equal opportunity difference**: Difference of true positive rates between the unprivileged and the privileged groups. Ideal value: 0 (-0.1 to 0.1 will consider as fair)
- **Average odds difference**: Average difference of false positive rate (false positives / negatives) and true positive rate (true positives / positives) between unprivileged and privileged groups. Ideal value: 0 (-0.1 to 0.1 will consider as fair)
- **Theil index**: Generalized entropy of benefit for all individuals in the dataset. It measures the inequality in benefit allocation for individuals. Ideal value: 0 (0 is the perfect fairness, there's no concrete interval to be considered as fair for this metric)
- **False negative rate difference**: Difference of false negative rate between unprivileged and privileged instances. Ideal value: 0 (-0.1 to 0.1 will consider as fair)

**Robustness Metrics**
- **Model accuracy on test data**: Amount of correct predictions using the original test data. Ideal value: 1
- **Model accuracy on adversarial samples**: Amount of correct predictions using the adversarial test samples. Ideal value: 1
- **Reduction in confidence**: Average amount of confidence score get reduced. Ideal value: 0
- **Average perturbation**: Average amount of [adversarial changes](https://en.wikipedia.org/wiki/Perturbation_theory) needed to make in order to fool the classifier. Ideal value: 1
