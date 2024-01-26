# Kubeflow Pipelines on Tekton

Project bringing Kubeflow Pipelines and Tekton together. The project is driven
according to this [design doc](http://bit.ly/kfp-tekton). The current code allows you run Kubeflow Pipelines with Tekton backend end to end.

* Create your Pipeline using Kubeflow Pipelines DSL, and compile it to Tekton
  YAML.
* Upload the compiled Tekton YAML to KFP engine (API and UI), and run end to end
  with logging and artifacts tracking enabled.
* In KFP-Tekton V2, the SDK compiler will generate the same intermediate representation as in the main Kubeflow pipelines SDK. All the Tekton related implementations are all embedded into the V2 backend API service.

For more details about the project please follow this detailed [blog post](https://developer.ibm.com/blogs/awb-tekton-optimizations-for-kubeflow-pipelines-2-0) . For the latest KFP-Tekton V2 implementation and [supported offerings](https://developer.ibm.com/articles/advance-machine-learning-workflows-with-ibm-watson-pipelines/), please follow our latest [Kubecon Talk](https://www.youtube.com/watch?v=ecx-yp4g7YU) and [slides](https://docs.google.com/presentation/d/1Su42ApXzZvVwhNSYRAk3bd0heHOtrdEX/edit?usp=sharing&ouid=103716780892927252554&rtpof=true&sd=true). For information on the KFP-Tekton V1 implementation, look at these [slides](https://www.slideshare.net/AnimeshSingh/kubeflow-pipelines-with-tekton-236769976) as well as this [deep dive presentation](https://www.youtube.com/watch?v=AYIeNtXLT_k) for demos.

## Architecture

We are currently using [Kubeflow Pipelines 1.8.4](https://github.com/kubeflow/pipelines/releases/tag/1.8.4) and
[Tekton >= 0.56.0](https://github.com/tektoncd/pipeline/releases/tag/v0.56.0)
in the master branch for this project.

For [Kubeflow Pipelines 2.0.5](https://github.com/kubeflow/pipelines/releases/tag/2.0.5) and
[Tekton >= 0.53.2](https://github.com/tektoncd/pipeline/releases/tag/v0.53.2)
integration, please check out the [kfp-tekton v2-integration](https://github.com/kubeflow/kfp-tekton/tree/v2-integration) branch and [KFP-Tekton V2 deployment](/guides/kfp_tekton_install.md#standalone-kubeflow-pipelines-v2-with-tekton-backend-deployment) instead.

![kfp-tekton](images/kfp-tekton.png)


Kubeflow Pipelines is a platform for building and deploying portable, scalable machine learning (ML) workflows. More architectural details about the Kubeflow Pipelines can be found on the [Kubeflow website](https://www.kubeflow.org/docs/components/pipelines/overview/).

The Tekton Pipelines project provides Kubernetes-style resources for declaring
CI/CD-style pipelines. Tekton introduces several [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)(CRDs) including Task, Pipeline, TaskRun, and PipelineRun. A PipelineRun represents a single running instance of a Pipeline and is responsible for creating a Pod for each of its Tasks and as many containers within each Pod as it has Steps. Please look for more details in the [Tekton repo](https://github.com/tektoncd/pipeline).

### Get Started using Kubeflow Pipelines on Tekton

[Install Kubeflow Pipelines with Tekton backend](/guides/kfp_tekton_install.md)

[KFP Tekton Pipelines User Guide](/guides/kfp-user-guide/README.md)

[Use KFP Tekton SDK](/sdk/README.md)

[Run Samples](/samples/README.md)

[Available KFP DSL Features](/sdk/FEATURES.md)

[Tekton Specific Features](/guides/advanced_user_guide.md)

### Development Guides

[Backend Developer Guide](/guides/developer_guide.md)

[SDK Developer Guide](/sdk/python/README.md)

[Compilation Tests Status Report](/sdk/python/tests/README.md)

### Design Guides

[Design Doc](http://bit.ly/kfp-tekton)

[KFP, Argo and Tekton Features Comparison](https://docs.google.com/spreadsheets/d/1LFUy86MhVrU2cRhXNsDU-OBzB4BlkT9C0ASD3hoXqpo/edit#gid=979402121)

### Community

[Kubeflow Slack](https://join.slack.com/t/kubeflow/shared_invite/zt-cpr020z4-PfcAue_2nw67~iIDy7maAQ)

### References

[Kubeflow and TFX Pipelines](/samples/kfp-tfx)

[Kubeflow and TFX Pipelines talk at Tensorflow World](https://www.slideshare.net/AnimeshSingh/hybrid-cloud-kubeflow-and-tensorflow-extended-tfx)
