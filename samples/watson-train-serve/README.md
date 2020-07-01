# Training and Serving Models with Watson Machine Learning

This pipeline runs training, storing and deploying a Tensorflow model with MNIST handwriting recognition using [IBM Watson Machine Learning](https://www.ibm.com/cloud/machine-learning) service. This example is originated from Kubeflow pipeline's [ibm-samples/watson](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/ibm-samples/watson) example.

## Prerequisites 
- Install [Kubeflow 1.0.2+](https://www.kubeflow.org/docs/started/getting-started/) and connect the cluster to the current shell with `kubectl`
- Install [Tekton 0.13.0](https://github.com/tektoncd/pipeline/releases/tag/v0.13.0)
    - For KFP, we shouldn't modify the default work directory for any component. Therefore, please run the below command to disable the [home and work directory overwrite](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#customizing-the-pipelines-controller-behavior) from Tekton default.
        ```shell
        kubectl patch cm feature-flags -n tekton-pipelines -p '{"data":{"disable-home-env-overwrite":"true","disable-working-directory-overwrite":"true"}}'
        ```
- Install [kfp-tekton](/sdk/README.md#steps) SDK and [Kubeflow pipeline with Tekton backend](/tekton_kfp_guide.md)

- Install [Watson Requirements](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/ibm-samples/watson#requirements) 

## Instructions

1. Compile the Watson ML pipeline
```shell
dsl-compile-tekton --py watson_train_serve_pipeline.py --output watson-train-server.yaml
```
2. If your default Kubeflow service account dosn't have edit permission, follow this [sa-and-rbac](/sdk/sa-and-rbac.md) to setup.

3. Next, upload the `watson-train-server.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline. Then, use the default pipeline variables except for these two variables. 

    `GITHUB_TOKEN`: your github token

    `CONFIG_FILE_URL`: your configuration file which stores the credential information, here is the example of [creds.ini file](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/ibm-samples/watson/credentials/creds.ini) 

