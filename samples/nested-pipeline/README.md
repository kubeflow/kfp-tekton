# Nested Pipeline example
This pipeline example demonstrates that within a pipeline, you can call another pipeline during execution, also known as nested pipeline. This example is originated from the kubeflow pipeline's [compose.py](https://github.com/kubeflow/pipelines/blob/master/sdk/python/tests/compiler/testdata/compose.py) example that is running with the KFP API. We have modified this example to run with the Tekton compiler and submit the pipeline directly to the Tekton engine.

This example contains two pipelines, `save-most-frequent` and `download-and-save-most-frequent`. `save-most-frequent` will take two inputs, one is a message string which you want to find the most frequent word in it, the other is the location/path you want to save in GCS, if you don't have GCS setup, this parameter will be ignored, and the result will not be saved. Both input fields are required. `download-and-save-most-frequent` will take two inputs, one is an URL with plain text, the second one is the location/path which you want to save the result in GCS. This pipeline will first download the text from the input URL, then call `save-most-frequent` pipeline. Both inputs are required.

## Prerequisites
- Install [Kubeflow 1.0.2+](https://www.kubeflow.org/docs/started/getting-started/) and connect the cluster to the current shell with `kubectl`
- Install [kfp-tekton](/sdk/README.md#steps) SDK

## Instructions

Once you have completed all the prerequisites for this example, you can first try out the `save-most-frequent` pipeline
```
dsl-compile-tekton --py compose.py --output pipeline.yaml --function save_most_frequent_word
kubectl apply -f pipeline.yaml
tkn pipeline start save-most-frequent --showlog
```
You will see the following params, specify the string and outputpath, both are required params, even you don't have GCS setup.
```
? Value for param `message` of type `string`? hello world hello
? Value for param `outputpath` of type `string`? result.txt

```
you will see the result
```
Waiting for logs to be available...
[get-frequent : get-frequent] hello

[save : save] {{inputs.parameters.get-frequent-word}}

```
To tryout the nested pipeline `download-and-save-most-frequent`:
```
dsl-compile-tekton --py compose.py --output pipeline.yaml --function download_save_most_frequent_word
kubectl apply -f pipeline.yaml
tkn pipeline start download-and-save-most-frequent --showlog
```
To give the input string and path
```
? Value for param `url` of type `string`? gs://ml-pipeline-playground/shakespeare1.txt
? Value for param `outputpath` of type `string`? res.txt
```
You will see the result
```
Waiting for logs to be available...
[download : download] With which he yoketh your rebellious necks Razeth your cities and subverts your towns And in a moment makes them desolate

[get-frequent : get-frequent] With which he yoketh your rebellious necks Razeth your cities and subverts your towns And in a moment makes them desolate
[get-frequent : get-frequent] 
[get-frequent : get-frequent] your

[save : save] Copying file:///tmp/results.txt...
/ [1 files][    6.0 B/    6.0 B]                                                              
[save : save] Operation completed over 1 objects/6.0 B.                                        

```
## Acknowledgements

The original [nested pipeline](https://github.com/kubeflow/pipelines/blob/master/sdk/python/tests/compiler/testdata/compose.py) from kubeflow/pipeline project.
