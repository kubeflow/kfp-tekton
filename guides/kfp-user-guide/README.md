# Kubeflow Pipelines with Tekton User Guide

This page introduces different ways to compile, upload, and execute Kubeflow Pipelines with Tekton backend. The usual flow for using the Kubeflow Pipeline is to compile the Kubeflow Pipeline Python DSL into a Tekton formatted file. Then upload the compiled file to the Kubeflow Pipeline platform. Lastly, execute the uploaded pipeline using the Kubeflow Pipeline backend engine. For starter, we recommend using the first method in each section.

In this tutorial, we use the below single step pipeline as our example
```python
from kfp import dsl
def echo_op():
    return dsl.ContainerOp(
        name='echo',
        image='busybox',
        command=['sh', '-c'],
        arguments=['echo "Got scheduled"']
    )

@dsl.pipeline(
    name='echo',
    description='echo pipeline'
)
def echo_pipeline(
):
    echo = echo_op()
```


## Table of Contents

  - [Compiling Pipelines](#compiling-pipelines)
    - [1. Compile Pipelines Using the `kfp_tekton.compiler.TektonCompiler` in Python](#1-compile-pipelines-using-the-kfp_tektoncompilertektoncompiler-in-python)
    - [2. Compile Pipelines Using the `dsl-compile-tekton` Bash Command Line Tool](#2-compile-pipelines-using-the-dsl-compile-tekton-bash-command-line-tool)
  - [Uploading Pipelines](#uploading-pipelines)
    - [1. Upload Pipelines with the Kubeflow Pipeline User Interface](#1-upload-pipelines-with-the-kubeflow-pipeline-user-interface)
    - [2. Upload Pipelines Using the `kfp_tekton.TektonClient` in Python](#2-upload-pipelines-using-the-kfp_tektontektonclient-in-python)
    - [3. Upload Pipelines Using the `kfp` Bash Command Line Tool](#3-upload-pipelines-using-the-kfp-bash-command-line-tool)
  - [Running Pipelines](#running-pipelines)
    - [1. Run Pipelines with the Kubeflow Pipeline User Interface](#1-run-pipelines-with-the-kubeflow-pipeline-user-interface)
    - [2. Run Pipelines Using the `kfp_tekton.TektonClient` in Python](#2-run-pipelines-using-the-kfp_tektontektonclient-in-python)
    - [3. Run Pipelines Using the `kfp` Bash Command Line Tool](#3-run-pipelines-using-the-kfp-bash-command-line-tool)
    - [4. Optional: Run Tekton Pipelines Without Using the Kubeflow Pipelines Engine](#4-optional-run-tekton-pipelines-without-using-the-kubeflow-pipelines-engine)
  - [Best Practices](#best-practices)
    - [Artifacts and Parameter output files for Tekton](#artifacts-and-parameter-output-files-for-tekton)
  - [Migration from Argo backend](#migration-from-argo-backend)
    - [Argo variables](#argo-variables)
    - [Absolute paths in commands](#absolute-paths-in-commands)
    - [Output artifacts and metrics](#output-artifacts-and-metrics)
    - [Default pipeline timeouts](#default-pipeline-timeouts)


## Compiling Pipelines

### 1. Compile Pipelines Using the `kfp_tekton.compiler.TektonCompiler` in Python

This is the recommended way to compile pipelines using Python. Here we will import the `TektonCompiler` class and use the `compile` function to compile the above `echo_pipeline` into a Tekton yaml called `echo_pipeline.yaml`. The output format can be renamed to one of the followings: `[.tar.gz, .tgz, .zip, .yaml, .yml]` 
```python
from kfp_tekton.compiler import TektonCompiler
TektonCompiler().compile(echo_pipeline, 'echo_pipeline.yaml')
```

In addition, we can put the above python code into a python `__main__` function. Then, the compilation can be done in terminal with a simple python command.
```shell
python echo_pipeline.py
```

### 2. Compile Pipelines Using the `dsl-compile-tekton` Bash Command Line Tool

The kfp-tekton SDK also comes with a bash command line tool for compiling Kubeflow pipelines. Here, we need to store the above `echo_pipeline` example into a python file called `echo_pipeline.py` and run the below bash command. The `--output` supports the following formats: `[.tar.gz, .tgz, .zip, .yaml, .yml]`
```shell
dsl-compile-tekton --py echo_pipeline.py  --output pipeline.yaml
```


## Uploading Pipelines

### 1. Upload Pipelines with the Kubeflow Pipeline User Interface

This is the recommended way to upload and manage pipeline using the Kubeflow pipeline web user interface. Go to the Kubeflow main dashboard(Endpoint of the istio-ingressgateway) and click on the **Pipelines** tab on the left panel. Then click on the **Upload pipeline button**.

![upload-button](images/upload-button.png)

Then, click on **Upload a file** and select our compiled pipeline file. Then click on **Upload** at the end to upload the pipeline.

![upload-page](images/upload-page.png)

Now, we should able to see the pipeline is being uploaded to the **Pipelines** page. 


### 2. Upload Pipelines Using the `kfp_tekton.TektonClient` in Python

To begin, we first need to declare our TektonClient:
- For single user:
```python
from kfp_tekton import TektonClient
# **Important**: Replace None to the KFP endpoint if the python session is not running on the Kubeflow cluster.
host = None
KUBEFLOW_PROFILE_NAME = None
client = TektonClient(host=host)
```

- For multi tenant:
1. `KUBEFLOW_PUBLIC_ENDPOINT_URL` - Kubeflow public endpoint URL. 
2. `SESSION_COOKIE` - A session cookie starts with authservice_session=. You can obtain it from your browser after authenticating with Kubeflow UI. Notice that this session cookie expires in 24 hours, so you need to obtain it again after cookie expired.
3. `KUBEFLOW_PROFILE_NAME` - Your Kubeflow profile/namespace name
```python
from kfp_tekton import TektonClient

KUBEFLOW_PUBLIC_ENDPOINT_URL = 'http://<Kubeflow_public_endpoint_URL>'
# this session cookie looks like "authservice_session=xxxxxxx"
SESSION_COOKIE = 'authservice_session=xxxxxxx'
KUBEFLOW_PROFILE_NAME = '<your-profile-name>'

client = TektonClient(
    host=f'{KUBEFLOW_PUBLIC_ENDPOINT_URL}/pipeline',
    cookies=SESSION_COOKIE
)
```

To upload the pipelines using Python, run the below code block inside the Python session. The below code block shows how to upload different versions of the pipeline using the Python client.
```python
import os

# Initial version of the compiled pipeline
pipeline_file_path = 'echo_pipeline.yaml'
pipeline_name = 'echo_pipeline'

# For the purpose of this tutorial, we will be using the same pipeline for both version.
pipeline_version_file_path = 'echo_pipeline.yaml'
pipeline_version_name = 'new_echo_pipeline'

# Upload initial version of the pipeline
pipeline_file = os.path.join(pipeline_file_path)
pipeline = client.pipeline_uploads.upload_pipeline(pipeline_file, name=pipeline_name)

# Upload new version of the pipeline
pipeline_version_file = os.path.join(pipeline_version_file_path)
pipeline_version = client.pipeline_uploads.upload_pipeline_version(pipeline_version_file,
                                                                   name=pipeline_version_name,
                                                                   pipelineid=pipeline.id)
```


### 3. Upload Pipelines Using the `kfp` Bash Command Line Tool

The kfp-tekton SDK also comes with a bash command line tool for uploading Kubeflow pipelines. Before running the below commands, we need to make sure our `kubectl` is connected to our Kubeflow cluster. Please be aware that the `kfp` CLI only works for single user mode.
```shell
kubectl get pods -n kubeflow | grep ml-pipeline
# ml-pipeline-fc87669c7-f98x4                                      1/1     Running   0          8d
# ml-pipeline-ml-pipeline-visualizationserver-569c95464-k9qcx      1/1     Running   0          33d
# ml-pipeline-persistenceagent-bb9986b46-l4kcx                     1/1     Running   0          8d
# ml-pipeline-scheduledworkflow-b959d6fd-bnc2w                     1/1     Running   0          33d
# ml-pipeline-ui-6f68595ff-tw295                                   1/1     Running   0          8d
# ml-pipeline-viewer-controller-deployment-7f65754d48-xz6jh        1/1     Running   0          33d
# ml-pipeline-viewer-crd-7bb858bb59-k9mz9                          1/1     Running   0          33d
# ml-pipeline-visualizationserver-d558b855-tdbwc                   1/1     Running   0          33d
```

To upload a new pipeline
```shell
kfp pipeline upload -p echo_pipeline echo_pipeline.yaml
# Pipeline 925415d5-18e9-4e08-b57f-3b06e3e54648 has been submitted

# Pipeline Details
# ------------------
# ID           925415d5-18e9-4e08-b57f-3b06e3e54648
# Name         echo_pipeline
# Description
# Uploaded at  2020-07-02T17:55:30+00:00
```

To upload a new version of an existing pipeline
```shell
kfp pipeline upload-version -p <existing_pipeline_id> -v new_pipeline_version echo_pipeline.yaml
# The new_pipeline_version version of the pipeline 925415d5-18e9-4e08-b57f-3b06e3e54648 has been submitted

# Pipeline Version Details
# --------------------------
# Pipeline ID   925415d5-18e9-4e08-b57f-3b06e3e54648
# Version Name  new_pipeline_version
# Uploaded at   2020-07-02T17:58:05+00:00
```


## Running Pipelines

### 1. Run Pipelines with the Kubeflow Pipeline User Interface

Once we have the pipeline uploaded, we can simply execute the pipeline by clicking on the pipeline name. Then click **Create run** on the pipeline page. 

![pipeline-page](images/pipeline-page.png)

Next, pick an experiment that this run will be associated with and click **Start** to execute the pipeline. **Picking an experiment is required for multi-tenant mode.**

![run-page](images/run-page.png)

Now, the pipeline is running and we can click on the pipeline run to view the execution graph.

![experiment-page](images/experiment-page.png)


### 2. Run Pipelines Using the `kfp_tekton.TektonClient` in Python

To begin, we first need to declare our TektonClient:
- For single user:
```python
from kfp_tekton import TektonClient
# **Important**: Replace None to the KFP endpoint if the python session is not running on the Kubeflow cluster.
host = None
KUBEFLOW_PROFILE_NAME = None
client = TektonClient(host=host)
```

- For multi tenant:
1. `KUBEFLOW_PUBLIC_ENDPOINT_URL` - Kubeflow public endpoint URL. 
2. `SESSION_COOKIE` - A session cookie starts with authservice_session=. You can obtain it from your browser after authenticated from Kubeflow UI. Notice that this session cookie expires in 24 hours, so you need to obtain it again after cookie expired.
3. `KUBEFLOW_PROFILE_NAME` - Your Kubeflow profile/namespace name
```python
from kfp_tekton import TektonClient

KUBEFLOW_PUBLIC_ENDPOINT_URL = 'http://<Kubeflow_public_endpoint_URL>'
# this session cookie looks like "authservice_session=xxxxxxx"
SESSION_COOKIE = 'authservice_session=xxxxxxx'
KUBEFLOW_PROFILE_NAME = '<your-profile-name>'

client = TektonClient(
    host=f'{KUBEFLOW_PUBLIC_ENDPOINT_URL}/pipeline',
    cookies=SESSION_COOKIE
)
```


The `TektonClient` can run pipelines using one of the below sources:

1. Python DSL source code
2. Compiled pipeline file
3. List of uploaded pipelines

To execute pipelines using the Python DSL source code, run the below code block in a Python session using the `echo_pipeline` example.
The `create_run_from_pipeline_func` takes the DSL source code to compile and run it directly using the Kubeflow pipeline API without
uploading it to the pipeline list. This method is recommended if we are doing some quick experiments without version control.

```python
# We can overwrite the pipeline default parameters by providing a dictionary of key-value arguments.
# If we don't want to overwrite the default parameters, then define the arguments as an empty dictionary.
arguments={}

client.create_run_from_pipeline_func(echo_pipeline, arguments=arguments, namespace=KUBEFLOW_PROFILE_NAME)
```

Alternatively, we can also run the pipeline directly using a pre-compiled file. 
```python
EXPERIMENT_NAME = 'Demo Experiments'
experiment = client.create_experiment(name=EXPERIMENT_NAME, namespace=KUBEFLOW_PROFILE_NAME)
run = client.run_pipeline(experiment.id, 'echo-pipeline', 'echo_pipeline.yaml')
``` 

Similarly, we can also run the pipeline from the list of uploaded pipelines using the same `run_pipeline` function. 

```python
EXPERIMENT_NAME = 'Demo Experiments'
experiment = client.create_experiment(name=EXPERIMENT_NAME, namespace=KUBEFLOW_PROFILE_NAME)

# Find the pipeline ID that we want to use.
client.list_pipelines()

run = client.run_pipeline(experiment.id, pipeline_id='925415d5-18e9-4e08-b57f-3b06e3e54648', job_name='echo_pipeline_run')
``` 


### 3. Run Pipelines Using the `kfp` Bash Command Line Tool

The kfp-tekton SDK also comes with a bash command line tool for running Kubeflow Pipelines. Before running the below commands, we need to make sure our `kubectl` is connected to our Kubeflow cluster. Please be aware that currently the `kfp` CLI only works for single user mode.
```shell
kubectl get pods -n kubeflow | grep ml-pipeline
# ml-pipeline-fc87669c7-f98x4                                      1/1     Running   0          8d
# ml-pipeline-ml-pipeline-visualizationserver-569c95464-k9qcx      1/1     Running   0          33d
# ml-pipeline-persistenceagent-bb9986b46-l4kcx                     1/1     Running   0          8d
# ml-pipeline-scheduledworkflow-b959d6fd-bnc2w                     1/1     Running   0          33d
# ml-pipeline-ui-6f68595ff-tw295                                   1/1     Running   0          8d
# ml-pipeline-viewer-controller-deployment-7f65754d48-xz6jh        1/1     Running   0          33d
# ml-pipeline-viewer-crd-7bb858bb59-k9mz9                          1/1     Running   0          33d
# ml-pipeline-visualizationserver-d558b855-tdbwc                   1/1     Running   0          33d
```

Then, we need to find the pipeline ID that we want to execute.
```shell
kfp pipeline list
# +--------------------------------------+-----------------------------------+---------------------------+
# | Pipeline ID                          | Name                              | Uploaded at               |
# +======================================+===================================+===========================+
# | 925415d5-18e9-4e08-b57f-3b06e3e54648 | echo_pipeline                     | 2020-07-02T17:55:30+00:00 |
# +--------------------------------------+-----------------------------------+---------------------------+
```

Next, submit the pipeline ID for execution
```shell
kfp run submit kfp run submit -e experiment-name -r run-name -p 925415d5-18e9-4e08-b57f-3b06e3e54648
# Creating experiment experiment-name.
# Run bb96363f-ec0d-4e5a-9ce9-f69a485c2d94 is submitted
# +--------------------------------------+----------+----------+---------------------------+
# | run id                               | name     | status   | created at                |
# +======================================+==========+==========+===========================+
# | bb96363f-ec0d-4e5a-9ce9-f69a485c2d94 | run-name |          | 2020-07-02T19:08:58+00:00 |
# +--------------------------------------+----------+----------+---------------------------+
```

Lastly, we can check the status of the pipeline execution
```shell
kfp run get bb96363f-ec0d-4e5a-9ce9-f69a485c2d94
# +--------------------------------------+----------+-----------+---------------------------+
# | run id                               | name     | status    | created at                |
# +======================================+==========+===========+===========================+
# | bb96363f-ec0d-4e5a-9ce9-f69a485c2d94 | run-name | Succeeded | 2020-07-02T19:08:58+00:00 |
# +--------------------------------------+----------+-----------+---------------------------+
```


### 4. Optional: Run Tekton Pipelines Without Using the Kubeflow Pipelines Engine

For all the compiled Kubeflow pipelines, we can extract them into `.yaml` format and run them directly on Kubernetes using the Tekton CRD.
However, this method should **only use for Tekton development** because Kubeflow Pipelines engine cannot track these pipelines that run directly on Tekton.

Assuming the `kubectl` is connected to a Kubernetes cluster with Tekton, we can simply run the apply command to run the Tekton pipeline.

```shell
kubectl apply -f echo_pipeline.yaml
```

Then we can view the status using the `kubectl describe` command.

```shell
kubectl describe pipelinerun echo
# Name:         echo
# Namespace:    default
# ...
# Events:
#   Type     Reason             Age                      From                 Message
#   ----     ------             ----                     ----                 -------
#   Warning  PipelineRunFailed  2s                       pipeline-controller  PipelineRun failed to update labels/annotations
#   Normal   Running            <invalid> (x11 over 2s)  pipeline-controller  Tasks Completed: 0, Incomplete: 1, Skipped: 0
#   Normal   Succeeded          <invalid>                pipeline-controller  Tasks Completed: 1, Skipped: 0
```

## Best Practices

### Artifacts and Parameter output files for Tekton
When developing a Kubeflow pipeline for the Tekton backend, please be aware that the files produced as artifacts and parameter outputs are carried to a volume mount path then get pushed to S3. It's not recommended to have volume mount on a container's root directory (`/`) because volume mount will overwrite all the files in the container path, including all the system files and binaries. 


Therefore, we have prohibited the kfp-tekton compiler from putting artifacts and parameter output files in the container's root directory. We recommend placing the output files inside a new directory under root to avoid this problem, such as `/tmp/`. The [condition](/samples/flip-coin/condition.py) example shows how the output files can be stored. Alternatively, you can learn [how to create reusable components](https://www.kubeflow.org/docs/pipelines/sdk/component-development/) in a component.yaml where you can avoid hard coding your output file path.

This also applies to the Argo backend with k8sapi and kubelet executors, and it's the recommended way to avoid the [race condition for Argo's PNS executor](https://github.com/argoproj/argo/issues/1256#issuecomment-494319015).

## Migration from Argo backend

### Argo variables

Variables like `{{workflow.uid}}` are currently not supported. See [the list of supported Argo variables](/sdk/FEATURES.md#variable-substitutions).

### Absolute paths in commands

Absolute paths in component commands (e.g. `python /app/run.py` instead of `python app/run.py`) are required unless Tekton has been patched for [disabling work directory overwrite](/sdk/python/README.md#tekton-cluster).
The patch happens automatically in case of whole Kubeflow deployment.

### Output artifacts and metrics

Output artifacts and metrics in Argo work implicitly by saving `/mlpipeline-metrics.json` or `/mlpipeline-ui-metadata.json`. 
In Tekton they need to be specified explicitly in the component. Also, `/` paths are not allowed (see [Artifacts and Parameter output files for Tekton](/guides/kfp-user-guide#artifacts-and-parameter-output-files-for-tekton). An example for Tekton:
```
output_artifact_paths={
    "mlpipeline-metrics": "/tmp/mlpipeline-metrics.json",
    "mlpipeline-ui-metadata": "/tmp/mlpipeline-ui-metadata.json",
}
```

### Default pipeline timeouts

Tekton pipelines have 1h timeout by default, Argo pipelines don't timeout by default. To disable timeouts by default, edit Tekton conifmap as follows:
```
kubectl edit cm config-defaults -n tekton-pipelines
# default-timeout-minutes: "0"
```

Pipeline-specific timeouts are possible by using `dsl.get_pipeline_conf().set_timeout(timeout)`.
