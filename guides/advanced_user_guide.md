# Advanced User Guide for Tekton Specific Features

This page is an advanced KFP-Tekton guide on how to use Tekton specific features such as [Tekton custom tasks](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#using-custom-tasks) on KFP. For general KFP usage, please visit the [kfp-user-guide](kfp-user-guide).

## Table of Contents

  - [Using Tekton Custom Task on KFP-Tekton](#using-tekton-custom-task-on-kfp-tekton)
    - [Basic Usage](#basic-usage)
    - [Defining Your Own Tekton Custom Task on KFP-Tekton](#defining-your-own-tekton-custom-task-on-kfp-tekton)
    - [Custom task with Conditions](#custom-task-with-conditions)
    - [Custom Task Limitations](#custom-task-limitations)

## Using Tekton Custom Task on KFP-Tekton

[Tekton Custom Tasks](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#using-custom-tasks)
can implement behavior that doesn't correspond directly to running a workload in a `Pod` on the cluster.
For example, a custom task might execute some operation outside of the cluster and wait for its execution to complete.

### Basic Usage
[The flip-coin pipeline using custom task](/samples/flip-coin-custom-task) example is a basic tutorial on how to use a pre-defined [CEL custom task](https://github.com/tektoncd/experimental/tree/main/cel) to replace the traditional KFP-Tekton conditions. Behind the scene, the pre-defined CEL custom task is written as the below containerOp in [/sdk/python/kfp_tekton/tekton.py](/sdk/python/kfp_tekton/tekton.py):
```python
def CEL_ConditionOp(condition_statement):
    '''A containerOp template for CEL and converts it into Tekton custom task
    during Tekton compiliation.
    Args:
        condition_statement: CEL expression statement using string and/or pipeline params.
    '''
    ConditionOp = dsl.ContainerOp(
            name="condition-cel",
            image=CEL_EVAL_IMAGE,
            command=["sh", "-c"],
            arguments=["--apiVersion", "cel.tekton.dev/v1alpha1",
                       "--kind", "CEL",
                       "--name", "cel_condition",
                       "--%s" % DEFAULT_CONDITION_OUTPUT_KEYWORD, condition_statement],
            file_outputs={DEFAULT_CONDITION_OUTPUT_KEYWORD: '/tmp/tekton'}
        )
    ConditionOp.add_pod_annotation("valid_container", "false")
    return ConditionOp
```

Here, the `apiVersion`, `kind`, and `name` are mandatory fields for all custom tasks. The `DEFAULT_CONDITION_OUTPUT_KEYWORD` field is a required argument only for running CEL custom task. The `CEL_EVAL_IMAGE` image is registered as a custom task in the KFP-Tekton compiler, so the backend knows this ContianerOp needs to be compiled differently.

### Defining Your Own Tekton Custom Task on KFP-Tekton
1. To define your own Tekton custom task on KFP-Tekton, first define a unique image name for classifying your containerOp definition as a custom task and update the list of registered custom task images.
    ```python
    MY_CUSTOM_TASK_IMAGE_NAME = "veryunique/image:latest"
    from kfp_tekton.tekton import TEKTON_CUSTOM_TASK_IMAGES
    TEKTON_CUSTOM_TASK_IMAGES = TEKTON_CUSTOM_TASK_IMAGES.append(MY_CUSTOM_TASK_IMAGE_NAME)
    ```
2. Define your own Tekton custom task as a containerOp following the below constraints:
    ```python
    ConditionOp = dsl.ContainerOp(
            name="any-name",
            image=MY_CUSTOM_TASK_IMAGE_NAME,
            command=["any", "command"],
            arguments=["--apiVersion", "custom_task_api_version",
                       "--kind", "custom_task_kind",
                       "--name", "custom_task_name",
                       "--taskSpec", {"task_spec_key": "task_spec_value"},
                       "--taskRef", {"task_ref_key": "task_ref_value"},
                       "--other_custom_task_argument_keys", custom_task_argument_values],
            file_outputs={"other_custom_task_argument_keys": '/anypath'}
        )
    # Annotation to tell the Argo controller that this containerOp is for specific Tekton runtime only.
    ConditionOp.add_pod_annotation("valid_container", "false")
    ```
    - **name**: Name for the custom task, it can be anything with a valid Kubernetes name.
    - **image**: Identifier for your custom task. This is the value you defined above in step 1. If you want to use a custom task with CEL condition support (such as using literal results from CEL custom task for Tekton whenExpression), then use `CEL_EVAL_IMAGE` instead.
    - **arguments**: Define the key value pairs for the custom task definition. Below are the mandatory fields
      - **--apiVersion**: Kubernetes API Version for your custom task CRD.
      - **--kind**: Kubernetes Kind for your custom task CRD.
      - **--name**: Kubernetes Resource reference name for your custom task CRD.
      - **--taskRef** (optional): Kubernetes Resource Spec for your custom task CRD. One of `--taskSpec` or `--taskRef` can be specified at a time.
        The value should be a Python Dictionary.
      - **--taskSpec** (optional): Kubernetes Resource Spec for your custom task CRD. This gets inlined in the pipeline.  One of `--taskSpec` or `--taskRef` can be specified at a time.
        Custom task controller should support [embedded spec](https://github.com/tektoncd/pipeline/blob/main/docs/runs.md#2-specifying-the-target-custom-task-by-embedding-its-spec).
        The value should be a Python Dictionary.
      - **Other arguments** (optional): Parameters for your custom task CRD inputs.

      Then, you can add any extra arguments to map as the custom task CRD's `spec.params` fields.
    - **--file_outputs**: List of Tekton results from the custom task. Since the custom task spec doesn't have Tekton results in static compile time, users need to explicitly define the custom task outputs to avoid mismatching between the custom task I/O.


### Custom task with Conditions
When using a custom task with conditions, the compiled workload is slightly different from Argo because we are not wrapping condition blocks as a sub-pipeline. The [flip-coin-custom-task](/samples/flip-coin-custom-task) example describes how custom tasks are used with KFP conditions.

The benefits for not having the sub-pipeline in Tekton allow us to represent the condition dependencies that are more aligned with the DSL. In KFP Argo, the [issue #5422](https://github.com/kubeflow/pipelines/issues/5422) describes how the Argo workflow misrepresent the DSL.

Condition DSL
```python
@dsl.pipeline(
    name='Conditional execution pipeline',
    description='Shows how to use dsl.Condition().'
)
def flipcoin_pipeline():
    flip = flip_coin_op()
    with dsl.Condition(flip.output == 'heads') as cond_a:
        random_num_head = random_num_op(0, 9)
        random_num_head2 = random_num_op(0, 9).after(random_num_head)

    with dsl.Condition(flip.output == 'tails') as cond_b:
        random_num_tail = random_num_op(10, 19)
        random_num_tail2 = random_num_op(10, 19).after(random_num_tail)

    print_output2 = print_op('after random_num_head random_num_tail').after(random_num_head, random_num_tail)
```

When compiled to Argo vs Tekton:

![condition-dependency](/images/condition-dependency.png)

In this case, the Argo workflow task C will always execute after the condition A,B blocks are done even if task A1 and B1 are skipped. Whereas the Tekton workflow task C only executes if task A1 and B1 are completed and will not execute Task C if task A1 and B1 are failed or skipped. So Tekton gives a closer representation of the DSL workflow behavior without using any sub-dag/sub-pipeline.


### Custom Task Limitations
Currently, custom tasks don't support timeout, retries, or any Kubernetes pod spec because these fields are not supported by the [Tekton community](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#limitations) and custom tasks may not need any pod to run their workloads. Therefore, there will not be any logs, pod references, or events for custom tasks in the KFP-Tekton API.
