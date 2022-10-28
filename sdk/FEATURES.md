# KFP-Tekton Compiler Features

This document describes the features supported by the _KFP-Tekton_ compiler.
With the current implementation, the _KFP-Tekton_ compiler is capable of
[compiling 80%](/sdk/python/tests/README.md) of approximately 90 sample
and test pipelines found in the KFP repository.

<!-- # Table of Contents -->

- [Pipeline DSL Features with Native Tekton Implementation](#pipeline-dsl-features-with-native-tekton-implementation)
    - [pod_annotations and pod_labels](#pod_annotations-and-pod_labels)
    - [Retries](#retries)
    - [Volumes](#volumes)
    - [Timeout for Tasks and Pipelines](#timeout-for-tasks-and-pipelines)
    - [RunAfter](#runafter)
    - [Input Parameters](#input-parameters)
    - [ContainerOp](#containerop)
    - [Affinity, Node Selector, and Tolerations](#affinity-node-selector-and-tolerations)
    - [ImagePullSecrets](#imagepullsecrets)
    - [Exit Handler](#exit-handler)
    - [Any Sequencer](#any-sequencer)
    - [Tekton Pipeline Variables](#tekton-pipeline-variables)
    - [Sidecars](#sidecars)
- [Pipeline DSL Features with a Custom Tekton Implementation](#pipeline-dsl-features-with-a-custom-tekton-implementation)
  - [Features with the Same Behavior as Argo](#features-with-the-same-behavior-as-argo)
    - [InitContainers](#initcontainers)
    - [Conditions](#conditions)
    - [ResourceOp and VolumeOp](#resourceop-and-volumeop)
    - [Output Parameters](#output-parameters)
    - [Input Artifacts](#input-artifacts)
    - [Output Artifacts](#output-artifacts)
    - [Caching](#caching)
    - [Pipeline Loops](#pipeline-loops)
    - [Enforce Artifact Tracking](#enforce-artifact-tracking)
  - [Features with Limitations](#features-with-limitations)
    - [Variable Substitutions](#variable-substitutions)


# Pipeline DSL Features with Native Tekton Implementation

Below are the features using Tekton's native support without any custom workaround.

### pod_annotations and pod_labels

`pod_annotations` and `pod_labels` are for assigning custom annotations or labels to a pipeline component. They are implemented with
Tekton's [task metadata](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#configuring-a-task) field under Tekton
Task. The [pipeline transformers](/sdk/python/tests/compiler/testdata/pipeline_transformers.py) example shows how to
 apply
custom annotations and labels to one or more components in the pipeline.

### Retries

Retry feature allows users to specify the number of times a particular component should retry its execution when it fails. It is
implemented with Tekton's [task spec](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md#using-the-retries-parameter)
under Tekton Pipeline. The [retry](/sdk/python/tests/compiler/testdata/retry.py) python test is an example of how to use
this feature.

### Volumes

Volumes are for mounting existing Kubernetes resources onto the components. It is implemented with Tekton's
[task volumes](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#specifying-volumes) feature under Tekton Task. The
[volume](/sdk/python/tests/compiler/testdata/volume.py) python test is an example of how to use this feature.

### Timeout for Tasks and Pipelines

Timeout can be used for setting the amount of time allowed on executing a component within the Pipeline or setting the amount of
time allowed on executing the whole pipeline. By default, the generated pipeline won't be timeout to simulate the same behavior
as Argo, but users can explicitly assign a timeout period on the task or pipeline level. The task timeout is implemented with Tekton's
[task failure timeout](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md#configuring-the-failure-timeout) under
Tekton Pipeline, and pipeline timeout is implemented with Tekton's
[pipeline failure timeout](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md#configuring-a-failure-timeout)
under Tekton PipelineRun. The [timeout](/sdk/python/tests/compiler/testdata/timeout.py) python test is an example of
how to use this feature.

If you want to use the
[Tekton global default timeout value](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md#configuring-a-failure-timeout)
for the generated pipeline, you can run `export TEKTON_GLOBAL_DEFAULT_TIMEOUT=true` to enable this feature.


### RunAfter

RunAfter is for indicating that a component must execute after one or more other components. It is implemented with Tekton's
[runAfter](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md#using-the-runafter-parameter) feature under Tekton
Pipeline. The [sequential](/sdk/python/tests/compiler/testdata/sequential.py) python test is an example of how to use this
feature.

### Input Parameters

Input Parameters are for passing pipeline parameters or other component outputs to the next running component. It is implemented
with Tekton's [parameters](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#specifying-parameters) features under Tekton
task. The [parallel_join](/sdk/python/tests/compiler/testdata/parallel_join.py) python test is an example of how to use this
feature.

### ContainerOp

ContainerOp defines the container spec for a pipeline component. It is implemented with Tekton's
[steps](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#defining-steps) features under Tekton task. The generated
Tekton task name is the same as the containerOp name whereas the step name is always called "main". The
[sequential](/sdk/python/tests/compiler/testdata/sequential.py) python test is an example of how to use this feature.

### Affinity, Node Selector, and Tolerations

Affinity, Node Selector, and Tolerations are Kubernetes specs for selecting which node should run the component based on
user-defined constraints. They are implemented with Tekton's [PipelineRunTaskSpec](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md#specifying-task-run-specs)
features under Tekton PipelineRun.
The [affinity](/sdk/python/tests/compiler/testdata/affinity.py),
[node_selector](/sdk/python/tests/compiler/testdata/node_selector.py), and
[tolerations](/sdk/python/tests/compiler/testdata/tolerations.py) Python tests are examples of how to use these features.

This feature has been implemented in Tekton version `0.13.0`.

### ImagePullSecrets

ImagePullSecret is a feature for the components to know which secret to use when pulling container images from private registries. It is implemented
with Tekton's [podTemplate](https://github.com/tektoncd/pipeline/blob/master/docs/podtemplates.md) field under Tekton
PipelineRun. The [imagepullsecrets](/sdk/python/tests/compiler/testdata/imagepullsecrets.py) Python test is an example of how to use this
feature.

This feature has been available since Tekton version `0.13.0`.

### Exit Handler

An _exit handler_ is a component that always executes, irrespective of success or failure,
at the end of the pipeline. It is implemented using Tekton's
[finally](https://github.com/tektoncd/pipeline/blob/v0.16.0/docs/pipelines.md#adding-finally-to-the-pipeline)
section under the Pipeline `spec`. An example of how to use an _exit handler_ can be found in
the [exit_handler](/sdk/python/tests/compiler/testdata/exit_handler.py) compiler test.

The `finally` syntax is supported since Tekton version `0.14.0`.

### Any Sequencer

When any one of the task dependencies completes successfully and the conditions meet, the dependent task will be started. Order of execution of the dependencies doesn’t matter, and the pipeline doesn't wait for all the task dependencies to complete before moving to the next step. Condition can be applied to enforce the task dependencies completes as expected. The condition expression should be the same format as is in Kubeflow ConditionOperator, and the result of containerOps can be used in expression. Notice the expression should only contain results from only one task because the purpose here is to check the simple condition for the task's output when a task complete. And also the operand in the expression should be int or string, other python types will be transferred to string automatically.

The Any Sequencer exits if all dependencies failed or skipped, or all conditions unmatched.

The exit status of Any Sequencer is configurable through parameters, `statusPath` must be specified for Any Sequencer to write status to, then use `skippingPolicy` and `errorPolicy`.
- skippingPolicy --- determines for the Any Sequencer reacts to no-dependency-condition-matching case. Values can be one of `skipOnNoMatch` or `errorOnNoMatch`, a status with value "Skipped" will be generated and the exit status will still be succeeded on `skipOnNoMatch`.
- errorPolicy --- the standard field, either `failOnError` or `continueOnError`. On `continueOnError`, a status with value "Failed" will be generated but the exit status will still be succeeded. For `Fail_on_error` the Any Sequencer should truly fail in the Tekton terms, as it does now.

Please follow the details of the implementation in the [design doc](https://docs.google.com/document/d/1oXOdiItI4GbEe_qzyBmMAqfLBjfYX1nM94WHY3EPa94/edit#heading=h.dt8bhna4spym).

For example:

```
from kfp_tekton.tekton import after_any

dsl.ContainerOp(
  ...
).apply(after_any([containerOps, conditionOp.output['result'] == 'true'], "any_sequencer_name"))
```

Please note that the service account of the `Any Sequencer` needs 'get' permission to watch the status of the specified `taskRun`.

### Tekton Pipeline Variables

Some [Tekton Pipeline Variables](https://github.com/tektoncd/pipeline/blob/master/docs/variables.md#variables-available-in-a-pipeline) that cannot be used in ContainerOps directly, and must be passed by `params`. The _KFP-Tekton_ compiler supports user using the variables in ContainerOps, the relative params will be added automatically when compiling. See the [example](python/tests/compiler/testdata/tekton_pipeline_variables.py).
Supported variables list:
```
$(context.pipeline.name)
$(context.pipelineRun.name)
$(context.pipelineRun.namespace)
$(context.pipelineRun.uid)
```

### Sidecars

Sidecars are containers that are executed in parallel with the main component within the same pod. It is implemented with Tekton's
[task sidecars](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#using-a-sidecar-in-a-task) features under Tekton Task. The
[sidecar](/sdk/python/tests/compiler/testdata/sidecar.py) python test is an example of how to use this feature.

**Note:** Tekton's current `Sidecar` implementation contains a bug.
Tekton uses a container image named `nop` to terminate `Sidecars`.
That image is configured by passing a flag to the Tekton controller.
If the configured `nop` image contains the exact command the `Sidecar`
was executing before receiving a "stop" signal, the `Sidecar` keeps
running, eventually causing the `TaskRun` to time out with an error.
For more information, see [Tekton issue 1347](https://github.com/tektoncd/pipeline/issues/1347).

# Pipeline DSL Features with a Custom Tekton Implementation

## Features with the Same Behavior as Argo

Below are the features that don't have one to one mapping to Tekton's native implementation, but the same behaviors can be replicated with
extra custom processing code or workaround within the compiler.

### InitContainers

InitContainers are containers that are executed before the main component within the same pod. Since Tekton already has the concept of task
steps, initContainers are placed as [steps](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#defining-steps) before the
main component task under Tekton Task. The [init_container](/sdk/python/tests/compiler/testdata/init_container.py) python test
is an example of how to use this feature.

### Conditions

Conditions are used for determining whether to execute certain components based on the output of the condition checks. In KFP Argo, each condition is represented as an Argo DAG template so it can be used as a dependency for other Argo templates. To replicate this in KFP Tekton, we put our condition into a dedicated Tekton task so that conditions can be treated as a dependency for other Tekton tasks. Another advantage of creating conditions using Tekton tasks is that we can have more flexible conditions such as comparing an integer and a float number, which currently is not available in Tekton. We are using the Tekton [when expression](https://github.com/tektoncd/pipeline/blob/master/docs/pipelines.md#guard-task-execution-using-whenexpressions) to check whether the condition task has succeeded or not. We created a custom python image to replicate the same condition checks that are in Argo and made it as the default in our compiler. The
[flip-coin](/samples/flip-coin) example demonstrates how to use multiple conditions within the same pipeline.

### ResourceOp and VolumeOp

[ResourceOp and VolumeOp](https://www.kubeflow.org/docs/pipelines/sdk/manipulate-resources/) are special operations for
creating Kubernetes resources on the pipeline cluster. ResourceOp is a basic operation for manipulating any Kubernetes resource. VolumeOps are operations for creating a unique Volume per pipeline and can be used as volume with any pipeline ContainerOp. Because Tekton has no support for these features natively, we have to implement our own custom task called kubectl-wrapper for creating these Kubernetes resources. The [resourceop_basic](/sdk/python/tests/compiler/testdata/resourceop_basic.py) and [volume_op](/sdk/python/tests/compiler/testdata/volume_op.py) python tests are examples of how to use these features.

### Output Parameters

Output parameters are a dictionary of string files that users can define as a component's outputs. In Tekton, we implemented this with Tekton [task results](https://github.com/tektoncd/pipeline/blob/master/docs/tasks.md#storing-execution-results) under Tekton Task. Since Tekton task results are only able to output parameters to its `/tekton/results` directory, we add an extra ending step to copy any non-Tekton user output to the `/tekton/results` directory. The [parallel_join](/sdk/python/tests/compiler/testdata/parallel_join.py) python test is an example of how to use this feature.

Due to the nature of Tekton using the Kubernetes Termination message to gather the task results, users only can have up to 4KB output parameters per task by default. However, if each of the output parameters is less than 2KB and users have a lot of output parameters, they can provide an estimated tekton result sizes for the output parameters to extend the Tekton limit. Below is the usage:

Add the annotation `tekton-result-sizes` to the defined KFP task as the following, the compiler will use these numbers to rearrange the tekton results into multiple steps in order to bypass the Tekton limit. [many_results](/sdk/python/tests/compiler/testdata/many_results.py) test is an example of how to use this feature.
```
    output_estimation_json = {'param1': 1500, 'param2': 700, 'param3': 500, 'param4': 900}
    print_task = print_op().add_pod_annotation('tekton-result-sizes', json.dumps(output_estimation_json))
```

### Input Artifacts

Input Artifacts in Kubeflow pipelines are used for passing raw text or local files as files placed in
the component pod. Since Input Artifacts can only be raw or in a compressed format as strings, we created a
[custom copy step](https://github.com/kubeflow/kfp-tekton/blob/7e4df54/sdk/python/kfp_tekton/compiler/_op_to_template.py#L237-L257)
for passing these strings as files before the main task is executed.
The [input_artifact_raw_value](/sdk/python/tests/compiler/testdata/input_artifact_raw_value.py) Python test
is an example of how to use this feature.

### Output Artifacts

Output Artifacts are files that need to be persisted to the default/destination object storage. Additionally, by default, all Kubeflow Pipeline 'Output Parameters' are also stored as output artifacts. Since Tekton deprecated pipelineResource and the recommended gsutil task is not capable of moving files to the minio object storage without proper DNS address, we decided to inject a step based on the [minio mc](https://github.com/minio/mc) image for moving output artifacts during Kubeflow Pipeline execution time.

It also includes several annontations, `tekton.dev/input_artifacts` and `tekton.dev/output_artifacts` are for metadata tracking, `tekton.dev/artifact_items` is to retain the artifact dependency information. Refer to the
[Tetkon Artifact design proposal](http://bit.ly/kfp-tekton) for more details.

The current implementation is relying on the existing KFP's minio setup for getting the default credentials. These default credentials can be updated
via a Kubernetes configmap.

### Caching

By default compiling a pipeline will add metadata annotations and labels so that results from tasks within a pipeline run can be re-used if that task is reused in a new pipeline run. This saves the pipeline run from re-executing the task when the results are already known. In order to disable caching, a label must be added to a task's metadata like in [this sample pipeline](./python/tests/compiler/testdata/cache.py#L39).

The specific annotations and labels that are added to the task spec metadata to enable caching are: `annotations={'tekton.dev/template': ""}` and `labels={'pipelines.kubeflow.org/cache_enabled': 'true', 'pipelines.kubeflow.org/pipelinename': '', 'pipelines.kubeflow.org/generation': ''}`.

### Pipeline Loops

PipelineLoops is a feature for running a component or a set of component tasks multiple times in a loop. Right now, Tekton supports loop pipeline/tasks via an implementation of [Tekton Custom Tasks Controller](https://github.com/tektoncd/community/blob/master/teps/0002-custom-tasks.md) named as "PipelineLoop". Please refer to the examples [here](/tekton-catalog/pipeline-loops/examples) to understand more details about the usage of loops.

By default, the SDK will not compile all the recursion loop resources in the pipelineRun annotations. If you want to apply the recursion loop resources together with pipelinerun as an admin, add the following code snippet before compiling the pipeline.
```python
import kfp_tekton
kfp_tekton.compiler.LOOP_RESOURCES_IN_SEPARATE_YAML=False
```

To use this feature, please ensure Tekton version >= v0.19, and "data.enable-custom-tasks" is "true" in feature-flags configmap:
`kubectl edit cm feature-flags -n tekton-pipelines`

To see how the Python SDK provides this feature, refer to the examples below:
- [loop_static](/sdk/python/tests/compiler/testdata/loop_static.py)
- [withparam_global](/sdk/python/tests/compiler/testdata/withparam_global.py)
- [withitem_nested](/sdk/python/tests/compiler/testdata/withitem_nested.py)
- [parallelfor_item_argument_resolving](/sdk/python/tests/compiler/testdata/parallelfor_item_argument_resolving.py)

### Enforce Artifact Tracking

If you pipelines require artifact tracking in order to run, enforce artfact tracking to be always on for your pipelines using one of the two ways.

Enable by adding a new pipeline annotation that enforce artifact tracking for the whole pipeline:
```python
import kfp_tekton
from kfp_tekton.compiler import TektonCompiler
pipeline_conf = kfp_tekton.compiler.pipeline_utils.TektonPipelineConf()
pipeline_conf.add_pipeline_annotation("tekton.dev/track_artifact", 'true')
TektonCompiler().compile(echo_pipeline, 'echo_pipeline.yaml', tekton_pipeline_conf=pipeline_conf)
```

Enable by adding a new task annotation that enforce artifact tracking for a specific task:
```python
task.add_pod_annotation("tekton.dev/track_step_artifact", "true")
```

## Features with Limitations

Below are the features that have certain limitation in the current Tekton release.

### Variable Substitutions

[Tracking issue #2322][VarSub]

[Here](https://github.com/tektoncd/pipeline/blob/master/docs/variables.md#variables-available-in-a-task) is the list of Tekton variables that will get substituted during pipeline execution. In addition, the compiler will automatically to map the below list of Argo variables to Tekton variables:
```
argo -> tekton
{{inputs.parameters.%s}} -> $(inputs.params.%s)
{{outputs.parameters.%s}} -> $(results.%s.path)
{{workflow.uid}} -> $(context.pipelineRun.uid)
{{workflow.name}} -> $(context.pipelineRun.name)
{{workflow.namespace}} -> $(context.pipelineRun.namespace)
{{workflow.parameters.%s}} -> $(params.%s)
```

[parallel_join_with_argo_vars](/sdk/python/tests/compiler/testdata/parallel_join_with_argo_vars.py) is an example of how Argo variables are
used and it can still be converted to Tekton variables with our Tekton compiler. However, other Argo variables will throw out an error because those Argo variables are very unique to Argo's pipeline system.


<!-- Issue and PR links-->

[ParallelFor]: https://github.com/tektoncd/pipeline/issues/2050
[VarSub]: https://github.com/tektoncd/pipeline/issues/1522
[Sidecars]: https://github.com/tektoncd/pipeline/issues/1347
