# Kubeflow Pipelines and Tekton
Experimental project to bring KFP and Tekton together, as well as curating Kubeflow samples in TFX, JenkinsX etc

## Tekton
The Tekton Pipelines project provides Kubernetes-style resources for declaring CI/CD-style pipelines.

Some tasks here wil invariably require contributions back to Tekton. Please follow the community guidelines there
https://github.com/tektoncd/pipeline

## Kubeflow Pipeline
Kubeflow pipelines are reusable end-to-end ML workflows built using the Kubeflow Pipelines SDK. 
https://github.com/kubeflow/pipelines

Since the work here will evolve from experimental towards a mature solution, we are keeping it separate to not affect the ongoing Kubeflow Pipeline changes. 

## CD Foundation

The work here is going to be tracked under the [CD Foundation MLOps Sig](https://cd.foundation/blog/2020/02/11/announcing-the-cd-foundation-mlops-sig/). If you are interested in joining, please see the [instructions here](https://github.com/cdfoundation/sig-mlops)

The work is going to be in accordance with these [design specification and decisions](http://bit.ly/kfp-tekton)

## Samples

1. [Kubeflow Pipelines-TFX Pipelines](/samples/kfp-tfx)
2. [Argo-Tekton Yaml for Flip Coin](/samples/kfp-tekton)
3. [KFP Compiler for basic sequential pipeline in Tekton](sdk/README.md)

## Evaluation Assets
1. [KFP, Argo and Tekton Comparision](https://docs.google.com/spreadsheets/d/1LFUy86MhVrU2cRhXNsDU-OBzB4BlkT9C0ASD3hoXqpo/edit#gid=979402121)
2. [Kubeflow Pipelines-TFX Pipelines Talk at Tensorflow World](https://www.slideshare.net/AnimeshSingh/hybrid-cloud-kubeflow-and-tensorflow-extended-tfx)
3. [Kubeflow Pipelines-TFX Pipelines RFC](https://docs.google.com/document/d/1_n3q0mNOr7gUSM04yaA0e5BO9RrS0Vkh1cNCyrB07WM/edit)
