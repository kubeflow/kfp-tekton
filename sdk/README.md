# Compiler for Tekton

There is an SDK for `Kubeflow Pipeline` for end users to define pipelines for AI and ML. The output of KFP SDK is yaml for [Argo](https://github.com/argoproj/argo).

Here we update the `Compiler` of SDK to generate `Tekton` yaml for a basic sequential pipeline. 

## Development Prerequisites
1. [`Python`](https://www.python.org/downloads/): Python 3.5 or later  
2. [`Conda`](https://docs.conda.io/en/latest/): Package, dependency and environment management for Python


## Steps

1. Setup Python env with Conda.

2. Clone the [Kubeflow Pipelines](https://github.com/kubeflow/pipelines)  

    `git clone https://github.com/kubeflow/pipelines`  
    `git checkout 0548e63f96d91c03bc213e0affaac7dfcd088c83`  

3. Copy `sdk/...` to overwrite the related path in `Kubeflow Pipelines`

4. Build the SDK

    - `cd $GOPATH`/src/github.com/kubeflow/pipelines/sdk/python  
    - `./build.sh`  
    - `pip install kfp.tar.gz`  

5. Comppile the sample pipeline

    - `cd sdk/samples`  
    - `dsl-compile --py ./sequential.py --output sequential.tar.gz`







