# Nested Pipeline example
This pipeline example demonstrates that within a pipeline, you can call another pipeline during execution, also known as nested pipeline. This example is originated from the kubeflow pipeline's [compose.py](https://github.com/kubeflow/pipelines/blob/master/sdk/python/tests/compiler/testdata/compose.py) example that is running with the KFP API. We have modified this example to run with the Tekton compiler and submit the pipeline directly to the Tekton engine.

This example contains two pipelines, `save-most-frequent` and `download-and-save-most-frequent`. `save-most-frequent` will take two inputs, one is a message string which you want to find the most frequent word in it, the other is the location/path you want to save in GCS, if you don't have GCS setup, this parameter will be ignored, and the result will not be saved. Both input fields are required. `download-and-save-most-frequent` will take two inputs, one is an URL with plain text, the second one is the location/path which you want to save the result in GCS. This pipeline will first download the text from the input URL, then call `save-most-frequent` pipeline. Both inputs are required.

## Prerequisites
- Install [Kubeflow 1.0.2+](https://www.kubeflow.org/docs/started/getting-started/) and connect the cluster to the current shell with `kubectl`
- Install [kfp-tekton](/sdk/README.md#steps) SDK and [Kubeflow pipeline with Tekton backend](/tekton_kfp_guide.md)

## Instructions

1. Once you have completed all the prerequisites for this example, you can first try out the `save-most-frequent` pipeline
```
dsl-compile-tekton --py compose.py --output pipeline.yaml --function save_most_frequent_word
```

Next, upload the `pipeline.yaml` file to the Kubeflow pipeline dashboard to run this pipeline.



2. To tryout the nested pipeline `download-and-save-most-frequent`:
```
dsl-compile-tekton --py compose.py --output pipeline.yaml --function download_save_most_frequent_word
```

Next, upload the `pipeline.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.


## Acknowledgements

The original [nested pipeline](https://github.com/kubeflow/pipelines/blob/master/sdk/python/tests/compiler/testdata/compose.py) from kubeflow/pipeline project.
