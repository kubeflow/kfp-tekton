# Pipeline with nested loops
This pipeline shows how to define loops and nested loops inside a loop. The nested loops feature demonstrates how Tekton can use the custom task feature to create multiple layers of sub-pipelines and loops on individual pipeline layers.

## Prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md) (Kubeflow 1.3+)

## Instructions

1. Compile the flip-coin pipeline using the compiler inside the python code. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `withitem_nested.py`.
    ```shell
    # Compile the python code
    python withitem_nested.py
    ```

    You will see 3 files, the pipeline file `withitem_nested.yaml` and two loop resource files `withitem_nested_pipelineloop_cr*.yaml`. Due to the limitation of the current Tekton API Spec. Loop spec only can be defined as separate resources. Thus, we need to register our loop resources with the commands below
    ```shell
    LOOP_RESOUCES=withitem_nested_pipelineloop_cr*
    awk 'FNR==1 && NR!=1 {print "---"}{print}' ${LOOP_RESOUCES} > loop_resources.yaml && kubectl apply -f loop_resources.yaml -n kubeflow
    ```

2. Upload the `withitem_nested.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline. Since Tekton PipelineRun API not yet support custom task spec, we will have to inspect the individual loop layers with `kubectl get pipelinerun -n kubeflow | grep my-pipeline`
