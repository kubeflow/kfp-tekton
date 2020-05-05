# Trusted AI framework integration

Artificial intelligence is becoming a crucial component of enterprises’ operations and strategy. In order to responsibly take advantage of AI today, we must figure out ways to instill transparency, explainability, fairness, and robustness into AI. In this example, we will be going over how to produce fairness and robustness metrics using the Trusted AI libraries from [AI Fairness 360](https://github.com/IBM/AIF360) and [Adversarial Robustness Toolbox](https://github.com/IBM/adversarial-robustness-toolbox).

This pipeline uses the [UTKface's aligned & cropped faces dataset](https://susanqq.github.io/UTKFace/) to train a gender classification model using the Katib engine. Once the training is completed, there will be two extra tasks that use the stored model and dataset to produce fairness and robustness metrics.

## Prerequisites 
- Install [Kubeflow 1.0.2+](https://www.kubeflow.org/docs/started/getting-started/) and connect the cluster to the current shell with `kubectl`
- Install [Tekton 0.11.3](https://github.com/tektoncd/pipeline/releases/tag/v0.11.3) and [Tekton CLI](https://github.com/tektoncd/cli)
    - For KFP, we shouldn't modify the default work directory for any component. Therefore, please run the below command to disable the [home and work directory overwrite](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#customizing-the-pipelines-controller-behavior) from Tekton default.
        ```shell
        kubectl patch cm feature-flags -n tekton-pipelines -p '{"data":{"disable-home-env-overwrite":"true","disable-working-directory-overwrite":"true"}}'
        ```
- Install [kfp-tekton](/sdk/README.md#steps) SDK

## Instructions

1. First, go to the Kubeflow dashboard and create a user namespace. The Kubeflow dashboard is the endpoint to your istio-ingressgateway. We will be using the namespace `anonymous` for this example.

2. Compile and apply the trusted-ai pipeline
```shell
dsl-compile-tekton --py trusted-ai.py --output trusted-ai.yaml
kubectl apply -f trusted-ai.yaml -n anonymous
```

3. Run the trusted-ai pipeline, click the `enter` key to use the default pipeline variables.
```shell
tkn pipeline start launch-katib-experiment -s default-editor -n anonymous --showlog
```

This pipeline will run for 10 to 15 minutes, then you should able to see the best hyperparameter tuning result at the end of the logs.
```
? Value for param `name` of type `string`? (Default is `trusted-ai`) trusted-ai
? Value for param `namespace` of type `string`? (Default is `anonymous`) anonymous
? Value for param `goal` of type `string`? (Default is `0.99`) 0.99
? Value for param `parallelTrialCount` of type `string`? (Default is `1`) 1
? Value for param `maxTrialCount` of type `string`? (Default is `1`) 1
? Value for param `experimentTimeoutMinutes` of type `string`? (Default is `60`) 60
? Value for param `deleteAfterDone` of type `string`? (Default is `True`) True
? Value for param `fgsm_attack_epsilon` of type `string`? (Default is `0.2`) 0.2
? Value for param `model_class_file` of type `string`? (Default is `PyTorchModel.py`) PyTorchModel.py
? Value for param `model_class_name` of type `string`? (Default is `ThreeLayerCNN`) ThreeLayerCNN
? Value for param `feature_testset_path` of type `string`? (Default is `processed_data/X_test.npy`) processed_data/X_test.npy
? Value for param `label_testset_path` of type `string`? (Default is `processed_data/y_test.npy`) processed_data/y_test.npy
? Value for param `protected_label_testset_path` of type `string`? (Default is `processed_data/p_test.npy`) processed_data/p_test.npy
? Value for param `favorable_label` of type `string`? (Default is `0.0`) 0.0
? Value for param `unfavorable_label` of type `string`? (Default is `1.0`) 1.0
? Value for param `privileged_groups` of type `string`? (Default is `[{'race': 0.0}]`) [{'race': 0.0}]
? Value for param `unprivileged_groups` of type `string`? (Default is `[{'race': 4.0}]`) [{'race': 4.0}]
? Value for param `loss_fn` of type `string`? (Default is `torch.nn.CrossEntropyLoss()`) torch.nn.CrossEntropyLoss()
? Value for param `optimizer` of type `string`? (Default is `torch.optim.Adam(model.parameters(), lr=0.001)`) torch.optim.Adam(model.parameters(), lr=0.001)
? Value for param `clip_values` of type `string`? (Default is `(0, 1)`) (0, 1)
? Value for param `nb_classes` of type `string`? (Default is `2`) 2
? Value for param `input_shape` of type `string`? (Default is `(1,3,64,64)`) (1,3,64,64)
Pipelinerun started: launch-ai-ethics-experiment-run-96lqr
Waiting for logs to be available...

[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Generating experiment template.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Creating kubeflow.org/experiments ai-ethics in namespace anonymous.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Created kubeflow.org/experiments ai-ethics in namespace anonymous.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments ai-ethics in namespace anonymous is Created.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments ai-ethics in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments ai-ethics in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments ai-ethics in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Current condition of kubeflow.org/experiments ai-ethics in namespace anonymous is Running.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:kubeflow.org/experiments ai-ethics in namespace anonymous has reached the expected condition: Succeeded.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Deleteing kubeflow.org/experiments ai-ethics in namespace anonymous.
[kubeflow-launch-experiment : kubeflow-launch-experiment] INFO:root:Deleted kubeflow.org/experiments ai-ethics in namespace anonymous.

[model-fairness-check : model-fairness-check] #### Plain model - without debiasing - classification metrics on test set
[model-fairness-check : model-fairness-check] metrics:  {'Classification accuracy': 0.8736901727555934, 'Balanced classification accuracy': 0.8736589340231593, 'Statistical parity difference': -0.08945831914198538, 'Disparate impact': 0.8325187064870843, 'Equal opportunity difference': -0.04179856216322675, 'Average odds difference': -0.028667217801755983, 'Theil index': 0.09022707769476579, 'False negative rate difference': 0.041798562163226846}

[adversarial-robustness-evaluation : adversarial-robustness-evaluation] metrics: {'model accuracy on test data': 0.8736901727555934, 'model accuracy on adversarial samples': 0.13565562163693004, 'confidence reduced on correctly classified adv_samples': 0.24403343492049726, 'average perturbation on misclassified adv_samples': 0.40323618054389954}
```

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
