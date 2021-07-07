# Training and Serving Models with Watson Machine Learning

This pipeline runs training, storing and deploying a Tensorflow model with MNIST handwriting recognition using [IBM Watson Studio](https://www.ibm.com/cloud/watson-studio/) service. This example is originated from Kubeflow pipeline's [ibm-samples/watson](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/ibm-samples/watson) example.

## Prerequisites 
- Install [KFP Tekton prerequisites](/samples/README.md)

- Install [Watson Requirements](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/ibm-samples/watson#requirements) 

## Instructions

1. Compile the Watson ML pipeline. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `watson_train_serve_pipeline.yaml`.
```shell
python watson_train_serve_pipeline.py
```
2. If your default Kubeflow service account dosn't have edit permission, follow this [sa-and-rbac](/sdk/sa-and-rbac.md) to setup.

3. Next, upload the `watson_train_serve_pipeline.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline. Then, use the default pipeline variables except for these two variables. 

    `GITHUB_TOKEN`: your github token

    `CONFIG_FILE_URL`: your configuration file which stores the credential information, here is the example of [creds.ini file](https://github.com/kubeflow/pipelines/blob/master/samples/contrib/ibm-samples/watson/credentials/creds.ini) 

