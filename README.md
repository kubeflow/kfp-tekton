# Kubeflow Pipelines with Tekton

Project to bring Kubeflow Pipelines and Tekton together. The work is being driven in accordance with this [design doc specifications](http://bit.ly/kfp-tekton). The work is split in phases. While the details of the phases are listed in the [design doc](http://bit.ly/kfp-tekton), the current code allows you run Kubeflow Pipelines with Tekton backend end to end.

* Craft your Pipelines using KFP DSL, and compile it to Tekton YAML. 
* Upload the compiled Tekton YAML to KFP UI and API engine, and run end to end with logging and artifacts tracking enabled.

## Tekton

The Tekton Pipelines project provides Kubernetes-style resources for declaring CI/CD-style pipelines. Tekton introduces
several new CRDs including Task, Pipeline, TaskRun, and PipelineRun. A PipelineRun represents a single running instance
of a Pipeline and is responsible for creating a Pod for each of its Tasks and as many containers within each Pod as it
has Steps. Please look for more details in [Tekton repo](https://github.com/tektoncd/pipeline).

## Kubeflow Pipeline with Tekton backend

We are currently using [Kubeflow Pipelines 0.5.1](https://github.com/kubeflow/pipelines/releases/tag/0.5.1) and
[Tekton 0.13.0](https://github.com/tektoncd/pipeline/releases/tag/v0.13.0) for this project.

![kfp-tekton](images/kfp-tekton.png)

## Installation Kubeflow Pipeline with Tekton

[Getting started with KFP Tekton deployment](tekton_kfp_guide.md)

### Getting Started
[Getting started with KFP Tekton Compiler SDK](/sdk/README.md)

### Developer Guide
[Developer Guide](/sdk/python/README.md) 

### Available Features and Implementation Details
[Available Features and Implementation Details](/sdk/FEATURES.md)

### Compiler Status Report
[Compilation Tests Status Report](/sdk/python/tests/README.md)

### Samples
[Samples being run end to end for verification](/samples/README.md)

### KFP, Argo and Tekton Features Comparison
[KFP, Argo and Tekton Features Comparison](https://docs.google.com/spreadsheets/d/1LFUy86MhVrU2cRhXNsDU-OBzB4BlkT9C0ASD3hoXqpo/edit#gid=979402121)

### Design Doc 
[Design Doc](http://bit.ly/kfp-tekton)

### CD Foundation

+ [CD Foundation MLOps Sig](https://cd.foundation/blog/2020/02/11/announcing-the-cd-foundation-mlops-sig/). 
+ [Instructions to join](https://github.com/cdfoundation/sig-mlops)

### Additional Reference Materials: KFP and TFX

+ [Kubeflow and TFX Pipelines](/samples/kfp-tfx)
+ [Kubeflow and TFX Pipelines talk at Tensorflow World](https://www.slideshare.net/AnimeshSingh/hybrid-cloud-kubeflow-and-tensorflow-extended-tfx)
