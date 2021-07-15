# Pipeline with nested loops
This pipeline shows how to define loops and nested loops inside a loop. The nested loops feature demonstrates how Tekton can use the custom task feature to create multiple layers of sub-pipelines and loops on individual pipeline layers.

## Prerequisites
- Install [KFP Tekton 0.9.0+](/guides/kfp_tekton_install.md#standalone-kubeflow-pipelines-with-tekton-backend-deployment)
- Install the [latest KFP SDK](https://github.com/kubeflow/kfp-tekton/tree/master/sdk#installation) (0.9.0+)

## Instructions

1. Compile the flip-coin pipeline using the compiler inside the python code. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `withitem_nested.py`.
    ```shell
    # Compile the python code
    python withitem_nested.py
    ```

2. Upload the `withitem_nested.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline. Since Tekton PipelineRun API not yet support custom task spec, we will have to inspect the individual loop layers with `kubectl get pipelinerun -n kubeflow | grep my-pipeline`
