# Kubeflow Pipelines with Tekton

Project to bring Kubeflow Pipelines and Tekton together. The project is split in phases and driven according to this [design doc](http://bit.ly/kfp-tekton). The current code allows you run Kubeflow Pipelines with Tekton backend end to end.

* Create your Pipeline using Kubeflow Pipelines DSL, and compile it to Tekton YAML. 
* Upload the compiled Tekton YAML to KFP engine (API and UI), and run end to end with logging and artifacts tracking enabled.

For more details about the project, including demos, please look at these [slides](https://www.slideshare.net/AnimeshSingh/kubeflow-pipelines-with-tekton-236769976) and the [deep dive presentation](https://www.youtube.com/watch?v=AYIeNtXLT_k).

## Tekton

The Tekton Pipelines project provides Kubernetes-style resources for declaring CI/CD-style pipelines. Tekton introduces
several new CRDs including Task, Pipeline, TaskRun, and PipelineRun. A PipelineRun represents a single running instance
of a Pipeline and is responsible for creating a Pod for each of its Tasks and as many containers within each Pod as it
has Steps. Please look for more details in [Tekton repo](https://github.com/tektoncd/pipeline).

## Kubeflow Pipeline with Tekton backend

We are currently using [Kubeflow Pipelines 0.5.1](https://github.com/kubeflow/pipelines/releases/tag/0.5.1) and
[Tekton 0.14.0](https://github.com/tektoncd/pipeline/releases/tag/v0.14.0) for this project.

![kfp-tekton](images/kfp-tekton.png)

### 1. Get Started using Kubeflow Pipelines with Tekton

1.1 [Install Kubeflow Pipelines with Tekton backend](tekton_kfp_guide.md)

1.2 [Use KFP Tekton SDK](/sdk/README.md)

1.3 [Run Samples](/samples/README.md)

1.4 [Available KFP DSL Features](/sdk/FEATURES.md)

### 2. Development Guides

2.1 [Developer Guide](/sdk/python/README.md) 

2.2 [Compilation Tests Status Report](/sdk/python/tests/README.md)

### 3. Design Guides

3.1 [Design Doc](http://bit.ly/kfp-tekton)

3.2 [KFP, Argo and Tekton Features Comparison](https://docs.google.com/spreadsheets/d/1LFUy86MhVrU2cRhXNsDU-OBzB4BlkT9C0ASD3hoXqpo/edit#gid=979402121)

### 4. Community

4.1 [CD Foundation MLOps Sig](https://cd.foundation/blog/2020/02/11/announcing-the-cd-foundation-mlops-sig/).

4.2 [Instructions to join](https://github.com/cdfoundation/sig-mlops)

### 5. References

5.1 [Kubeflow and TFX Pipelines](/samples/kfp-tfx)

5.2 [Kubeflow and TFX Pipelines talk at Tensorflow World](https://www.slideshare.net/AnimeshSingh/hybrid-cloud-kubeflow-and-tensorflow-extended-tfx)
