# Compiler for Tekton

There is an [SDK](https://www.kubeflow.org/docs/pipelines/sdk/sdk-overview/) 
for `Kubeflow Pipeline` for end users to define end to end machine learning and data pipelines.
The output of the KFP SDK compiler is YAML for [Argo](https://github.com/argoproj/argo).

Here we update the `Compiler` of the KFP SDK to generate `Tekton` YAML for a basic pipeline with parallal and sequential steps. Please go through these steps to ensure you are setup properly to use the compiler.


## Development Prerequisites

1. [`Python`](https://www.python.org/downloads/): Python 3.5 or later  
2. [`Conda`](https://docs.conda.io/en/latest/) or Python 
   [virtual environment](https://packaging.python.org/guides/installing-using-pip-and-virtual-environments/): 
   Package, dependency and environment management for Python


## Steps

1. Clone the kfp-tekton repo:
    - `git clone https://github.com/kubeflow/kfp-tekton.git`
    - `cd kfp-tekton`


2. Setup Python environment with Conda or a Python virtual environment:

    - `python3 -m venv .venv`
    - `source .venv/bin/activate`

3. Build the compiler:

    - `pip install -e sdk/python`

4. Run the compiler tests (optional):

    - `./sdk/python/tests/run_tests.sh`

5. Compile the sample pipeline:

    - `cd sdk/samples`  
    - `dsl-compile-tekton --py parallel_join.py --output pipeline.yaml`
    
6. Run the sample pipeline on a Tekton cluster:

    - `kubectl apply -f pipeline.yaml`
    - `tkn pipeline start parallel-pipeline`

   You should see messages asking for default URLs like below. Press `enter` and take the defaults
    ```bash
      ? Value for param `url1` of type `string`? (Default is `gs://ml-pipeline-playgro 
      ? Value for param `url1` of type `string`? (Default  is `gs://ml-pipeline-playground/shakespeare1.txt`) gs://ml-pipeline-
      playground/shakespeare1.txt
      ? Value for param `url2` of type `string`? (Default is `gs://ml-pipeline-playgro? Value for param `url2` of type `string`? (Default 
      is  `gs://ml-pipeline-playground/shakespeare2.txt`) gs://ml-pipeline-playground/shakespeare2.txt
 
      Pipelinerun started: parallel-pipeline-run-th4x6
      
      In order to track the pipelinerun progress run:
      tkn pipelinerun logs parallel-pipeline-run-th4x6 -f -n default
    ```
 7.  Run the command output in previous step to track the logs of the running Tekton Pipeline.

      ```bash
      tkn pipelinerun logs parallel-pipeline-run-th4x6 -f -n default
      ```
   
      You should see an output similar to the one below
      
      ```bash
      [gcs-download-2 : gcs-download-2] I find thou art no less than fame hath bruited And more than may be gatherd by thy shape Let my    
      presumption not provoke thy wrath
      [gcs-download : gcs-download] With which he yoketh your rebellious necks Razeth your cities and subverts your towns And in a moment         makes them desolate
      [echo : echo] Text 1: With which he yoketh your rebellious necks Razeth your cities and subverts your towns And in a moment makes           them desolate
      [echo : echo] Text 2: I find thou art no less than fame hath bruited And more than may be gatherd by thy shape Let my presumption not 
      provoke thy wrath
      ```

## Tested Versions

 - Python: `3.7.5`
 - Kubeflow Pipelines: [`0.2.2`](https://github.com/kubeflow/pipelines/releases/tag/0.2.2)
 - Tekton: [`0.11.0`](https://github.com/tektoncd/pipeline/releases/tag/v0.11.0-rc1)
 
