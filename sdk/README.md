# Compiler for Tekton

As we know, there is a SDK of `kubeflow pipeline` for end user to define pipelines for machine learning(ML). Actually the output stuff fo the SDK is yaml for [Argo](https://github.com/argoproj/argo).

In this place, we try to update the `Comiler` of SDK to generate `Tekton` stuff.

## Development Prerequisites
1. [`Python`](https://www.python.org/downloads/): Python 3.5 or later  
2. [`Conda`](https://docs.conda.io/en/latest/): Package, dependency and environment management for Python


## Take a try  

1. Setup Python env with Conda.

2. Clone the [Kubeflow Pipelines](https://github.com/kubeflow/pipelines)  

    `git clone https://github.com/kubeflow/pipelines`  
    `git checkout 0548e63f96d91c03bc213e0affaac7dfcd088c83`  

3. Copy `sdk/...` to overwrite related path in `Kubeflow Pipelines`

4. Build the SDK

    - `cd $GOPATH`/src/github.com/kubeflow/pipelines/sdk/python  
    - `./build.sh`  
    - `pip install kfp.tar.gz`  

5. Complier a sample

    - `cd sdk/samples`  
    - `dsl-compile --py ./sequential.py --output sequential.tar.gz`


## Description

For now, it's just tiny change to the original Compiler only support make a `sequential` case.






