# Compiler for Tekton

There is an [SDK](https://www.kubeflow.org/docs/pipelines/sdk/sdk-overview/) for `Kubeflow Pipeline` for end users to
define pipelines for AI and ML. The output of the KFP SDK compiler is YAML for [Argo](https://github.com/argoproj/argo).

Here we update the `Compiler` of the KFP SDK to generate `Tekton` YAML for a basic sequential pipeline. 


## Development Prerequisites

1. [`Python`](https://www.python.org/downloads/): Python 3.5 or later  
2. [`Conda`](https://docs.conda.io/en/latest/) or Python 
   [virtual environment](https://packaging.python.org/guides/installing-using-pip-and-virtual-environments/): 
   Package, dependency and environment management for Python


## Steps

1. Setup Python environment with Conda or a Python virtual environment:

    - `python3 -m venv .venv`
    - `source .venv/bin/activate`

2. Clone the [Kubeflow Pipelines](https://github.com/kubeflow/pipelines) 
   repository version `0.2.2` to local `build` directory:

    - `git clone --single-branch -b 0.2.2 --depth 1 https://github.com/kubeflow/pipelines build/kubeflow/pipelines`

3. Copy the KFP-Tekton compiler files from `sdk/...` to overwrite the related
   KFP SDK path in the local `build` directory:

    - `cp -rf sdk/python build/kubeflow/pipelines/sdk`

4. Build the SDK:

    - `cd build/kubeflow/pipelines/sdk/python`
    - `./build.sh`
    - `pip install kfp.tar.gz`
    - `cd -`

5. Compile the sample pipeline:

    - `cd sdk/samples/sequential`  
    - `dsl-compile --py ./sequential.py --output sequential.tar.gz`
    - `tar -xzvf sequential.tar.gz`
    
6. Run the sample pipeline on a Tekton cluster:

    - `kubectl apply -f pipeline.yaml`
    - `tkn task start sequential-pipeline`
