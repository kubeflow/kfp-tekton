# Compiler for Tekton

There is an [SDK](https://www.kubeflow.org/docs/pipelines/sdk/sdk-overview/) 
for `Kubeflow Pipeline` for end users to define pipelines for AI and ML. 
The output of the KFP SDK compiler is YAML for [Argo](https://github.com/argoproj/argo).

Here we update the `Compiler` of the KFP SDK to generate `Tekton` YAML for 
a basic sequential pipeline. 


## Development Prerequisites

1. [`Python`](https://www.python.org/downloads/): Python 3.5 or later  
2. [`Conda`](https://docs.conda.io/en/latest/) or Python 
   [virtual environment](https://packaging.python.org/guides/installing-using-pip-and-virtual-environments/): 
   Package, dependency and environment management for Python


## Steps

1. Setup Python environment with Conda or a Python virtual environment:

    - `python3 -m venv .venv`
    - `source .venv/bin/activate`

2. Build the compiler:

    - `pip install sdk/python`

3. Run the compiler tests (optional):

    - `./sdk/python/tests/run_tests.sh`

4. Compile the sample pipeline:

    - `cd sdk/samples/sequential`  
    - `dsl-compile-tekton --py ./sequential.py --output pipeline.yaml`
    
5. Run the sample pipeline on a Tekton cluster:

    - `kubectl apply -f pipeline.yaml`
    - `tkn task start sequential-pipeline`
