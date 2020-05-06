# Watson Train and Server.

This pipeline runs training, storing and deploying a Tensorflow model with MNIST handwriting recognition using [IBM Watson Machine Learning](https://www.ibm.com/cloud/machine-learning) service.

## Prerequisites 
- Install [Kubeflow 1.0.2+](https://www.kubeflow.org/docs/started/getting-started/) and connect the cluster to the current shell with `kubectl`
- Install [Tekton 0.11.3](https://github.com/tektoncd/pipeline/releases/tag/v0.11.3) and [Tekton CLI](https://github.com/tektoncd/cli)
    - For KFP, we shouldn't modify the default work directory for any component. Therefore, please run the below command to disable the [home and work directory overwrite](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#customizing-the-pipelines-controller-behavior) from Tekton default.
        ```shell
        kubectl patch cm feature-flags -n tekton-pipelines -p '{"data":{"disable-home-env-overwrite":"true","disable-working-directory-overwrite":"true"}}'
        ```
- Install [kfp-tekton](/sdk/README.md#steps) SDK

- Install [Watson Requirements](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/ibm-samples/watson#requirements) 

## Instructions

1. Compile and apply the trusted-ai pipeline
```shell
dsl-compile-tekton --py watson_train_serve_pipeline.py --output watson-train-server.yaml
kubectl apply -f watson-train-server.yaml 
```
2. If your service account hasn't have RBAC, follow this [sa-and-rbac](/sdk/sa-and-rbac.md) to setup.

3. Run the kfp-on-wml-training pipeline, click the `enter` key to use the default pipeline variables except these two variables,
`GITHUB_TOKEN`: your github token
`CONFIG_FILE_URL`: your configuration file which stores the credential information, here is the example of [creds.ini file](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/ibm-samples/watson/credentials/creds.ini) 

```shell
tkn pipeline start kfp-on-wml-training --showlog
```

This pipeline will run for 5 to 10 minutes, then you should able to see the result at the end of the logs.
```
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] #######################################################################################
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] Synchronous deployment creation for uid: '0f389a61-a48d-459b-9435-da479011bb53' started
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] #######################################################################################
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] initializing
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] ready
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] ------------------------------------------------------------------------------------------------
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] Successfully finished deployment creation, deployment_uid='0109b840-27b1-4d0d-82ab-faa86f1ce6fa'
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] ------------------------------------------------------------------------------------------------
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] deployment_uid:  0109b840-27b1-4d0d-82ab-faa86f1ce6fa
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] Scoring result: 
[deploy-model-watson-machine-learning : deploy-model-watson-machine-learning] {'predictions': [{'values': [5, 4]}]}

```

