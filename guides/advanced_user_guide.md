# Advanced User Guide for Tekton Specific Features

This page is an advanced KFP-Tekton guide on how to use Tekton specific features such as [Tekton custom tasks](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#using-custom-tasks) on KFP. For general KFP usage, please visit the [kfp-user-guide](kfp-user-guide).

## Table of Contents

  - [Using Tekton Custom Task on KFP-Tekton](#using-tekton-custom-task-on-kfp-tekton)
    - [Basic Usage](#basic-usage)
    - [Defining Your Own Tekton Custom Task on KFP-Tekton](#defining-your-own-tekton-custom-task-on-kfp-tekton)
    - [Custom task with Conditions](#custom-task-with-conditions)
    - [Custom Task Limitations](#custom-task-limitations)
    - [Custom Task for OpsGroup](#custom-task-for-opsgroup)
  - [Tekton Pipeline Config for PodTemplate](#tekton-pipeline-config-for-podtemplate)
  - [Tekton Feature Flags](#tekton-feature-flags)

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
            arguments=["--apiVersion", "custom_task_api_version",
                       "--kind", "custom_task_kind",
                       "--name", "custom_task_name",
                       "--taskSpec", {"task_spec_key": "task_spec_value"},
                       "--taskRef", {"task_ref_key": "task_ref_value"}],
            command=["--other_custom_task_argument_keys", custom_task_argument_values],
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
        Custom task controller should support [embedded spec](https://github.com/tektoncd/pipeline/blob/main/docs/customruns.md#2-specifying-the-target-custom-task-by-embedding-its-spec).
        The value should be a Python Dictionary.
    - **command**: Define the key value pairs for the custom task input parameters.
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


### Custom Task for OpsGroup
You can also use Custom Task for an OpsGroup. An OpsGroup is used to represent a logical group of ops
and group of OpsGroups. We provide a base class `AddOnGroup` under `kfp_tekton.tekton` for you to
implement your OpsGroup. You can define a custom OpsGroup like this:
```python
from typing import Dict
from kfp import dsl
from kfp_tekton.tekton import AddOnGroup
class MyOpsGroup(AddOnGroup):
  """A custom OpsGroup which maps to a custom task"""

  def __init__(self, params: Dict[str, dsl.PipelineParam] = {}):
    super().__init__(kind='CustomGroup',
        api_version='custom.tekton.dev/v1alpha1',
        params=params)
```

For `params` argument, it's used to generate the spec.params for the Custom Task. For intermediate params
which are only used by downstream Ops/OpsGroup but no need to show up in spec.params, you need to use
`AddOnGroup.create_internal_param` function to create a PipelineParam and store it to params Dict.

Then you can use the custom OpsGroup to contain arbitrary ops and even other OpsGroup as well. For example:
```python
@dsl.pipeline(
    name='addon-sample',
    description='Addon sample'
)
def addon_example(url: str = 'gs://ml-pipeline-playground/shakespeare1.txt'):
    """A sample pipeline showing AddOnGroups"""

    echo_op('echo task!')

    with MyOpsGroup():
        download_task = gcs_download_op(url)    # op 1
        echo_op(download_task.outputs['data'])  # op 2
```
The way to use the custom OpsGroup depends on how you compose your custom OpsGroup which extends
the base class AddOnGroup. You can use the `with` syntax like the example above. Or you can implement your
custom OpsGroup as a function decorator and covert a function into custom OpsGroup. The generated Task YAML
is based on the following information: `kind`, `apiVersion`, `is_finally`, ops, OpsGroup, etc.
For ops and OpsGroup belong to the custom OpsGroup, they become individual task in the `tasks` list.
Below is an example of the generated Task YAML for the custom OpsGroup: `MyOpsGroup` from the example above.
```yaml
  - name: addon-group-1
    params:
    - name: url
      value: $(params.url)
    taskSpec:
      apiVersion: custom.tekton.dev/v1alpha1
      kind: CustomGroup
      spec:
        pipelineSpec:
          params:
          - name: url
            type: string
          tasks:
          - name: gcs-download
            ......
          - name: echo-2
            ......
```
You can override the `post_task_spec` or `post_params` functions to manipulate the Task YAML or `spec.params` if needed. The custom OpsGroup
could be assigned to [`finally` section of Tekton Pipeline](https://tekton.dev/docs/pipelines/pipelines/#adding-finally-to-the-pipeline)
by assigning `is_finally=Ture` when constructing AddOnGroup. In this case, the custom OpsGroup can only be
used as a root OpsGroup in a pipeline. It can't be under other OpsGroup.


## Tekton Pipeline Config for PodTemplate
Currently users can explicitly setup Security Context and Auto Mount Service Account Token at the pipeline level using Tekton Pipleine Config.
Below are the usages and input types:
- set_security_context() - InputType: `V1SecurityContext`
- set_automount_service_account_token() - InputType: `Bool`
- add_pipeline_env() - InputType: name `str`, value `str`
- set_pipeline_env() - InputType: `Dict`
- add_pipeline_workspace() - InputType: workspace_name `str`, volume `V1Volume` (optional), volume_claim_template_spec `V1PersistentVolumeClaimSpec` (optional), path_prefix `str`
- set_generate_component_spec_annotations() - InputType: `Bool`
<<<<<<< HEAD
=======
- set_condition_image_name() - InputType: `str`
- set_bash_image_name() -  InputType: `str`
>>>>>>> 972c8817f (feat(sdk): add bash script name config (#1334))

```python
from kfp_tekton.compiler.pipeline_utils import TektonPipelineConf
from kubernetes.client import V1SecurityContext
from kubernetes.client.models import V1Volume, V1PersistentVolumeClaimVolumeSource, \
    V1PersistentVolumeClaimSpec, V1ResourceRequirements
pipeline_conf = TektonPipelineConf()
pipeline_conf.set_security_context(V1SecurityContext(run_as_user=0)) # InputType: V1SecurityContext
pipeline_conf.set_automount_service_account_token(False) # InputType: Bool
pipeline_conf.add_pipeline_env('ENV_NAME', 'VALUE') # InputType: name `str`, value `str`
pipeline_conf.add_pipeline_workspace(workspace_name="new-ws", volume=V1Volume(
    name='data',
    persistent_volume_claim=V1PersistentVolumeClaimVolumeSource(
        claim_name='data-volume')
), path_prefix='artifact_data/') # InputType: workspace_name `str`, volume `V1Volume`, path_prefix `str`
pipeline_conf.add_pipeline_workspace(workspace_name="new-ws-template",
    volume_claim_template_spec=V1PersistentVolumeClaimSpec(
        access_modes=["ReadWriteOnce"],
        resources=V1ResourceRequirements(requests={"storage": "30Gi"})
)) # InputType: workspace_name `str`, volume_claim_template_spec `V1PersistentVolumeClaimSpec`
pipeline_conf.set_generate_component_spec_annotations(True) # InputType: Bool
self._test_pipeline_workflow(test_pipeline, 'test.yaml', tekton_pipeline_conf=pipeline_conf)
```

If you want to use the configured workspaces inside the task spec, you can add the `workspaces` annotation below to have the workspaces volume mounted in your tasks. From there, you can use the Tekton context variable to retrieve the Tekton workspace path.

```python
import json
echo = echo_op() # tasks in the pipeline function
workspace_json = {'new-ws': {"readOnly": True}}
echo.add_pod_annotation('workspaces', json.dumps(workspace_json))
```

For more details on how this can be used in a real pipeline, visit the [Tekton Pipeline Conf example](/sdk/python/tests/compiler/testdata/tekton_pipeline_conf.py).

## Tekton Feature Flags
Tekton provides some features that you can configure via `feature-flgas` configmap under `tekton-pipelines`
namespace. Use these tekton features may impact kfp-tekton backend. Here is the list of features that
impact kfp-tekton backend:
- enable-custom-tasks: You can turn on/off custom task support by setting its value to true/false.
  The default value for kfp-tekton deployment is `true`. If you are using custom tasks and set the flag value
  to `false`, the pipeline will be in the running state until timeout. Because custom tasks inside the pipeline
  are not able to be handled properly.
- embedded-status: You can find details for this feature flag [here](https://github.com/tektoncd/community/blob/main/teps/0100-embedded-taskruns-and-runs-status-in-pipelineruns.md).
  The default value for kfp-tekton deployment is `full`, which stores all TaskRuns/Runs statuses under PipelineRun's status.
  kfp-tekton backend also supports the `minimal` setting, which only records the list of TaskRuns/Runs under PipelineRun's status.
  In this case, statuses of TaskRuns/Runs only exist in their own CRs. kfp-tekton backend retrieves statuses of TaskRuns/Runs
  from individual CR, aggregates, and stores them into the backend storage.