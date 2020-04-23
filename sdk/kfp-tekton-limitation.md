
# Limitations

- [Overview](#overview)
  - [Affinity, Node Selector, and Tolerations](#affinity-node-selector-tolerations)
  - [Sidecars](#sidecars)
  - [Store the *secrets* which need to pull images](#store-secrets-to-pull-images)

## Overview

During the development and testing, we noticed some limitations for the kfp-tekton project. Here we list those limitations. We are working on those limitations, some limitations are from the tekton pipeline projects, we have opened issues there to track the progress.

<a name="affinity-node-selector-tolerations" />

### Affinity, Node Selector, and Tolerations

Please be aware that defined Affinity, Node Selector, and Tolerations are applied to all the tasks in the same pipeline because there's only one podTemplate allowed in each pipeline.

<a name="sidecars" />

### Sidecars

When you run kfp-tekton pipeline with sidecars, you may notice a completed pod and a sidercar pod with Error. There are known issues with the existing implementation of sidecars:

When the nop image does provide the sidecar's command, the sidecar will continue to run even after nop has been swapped into the sidecar container's image field. See the issue tracking this bug for the issue tracking this bug. Until this issue is resolved the best way to avoid it is to avoid overriding the nop image when deploying the tekton controller, or ensuring that the overridden nop image contains as few commands as possible.

kubectl get pods will show a Completed pod when a sidecar exits successfully but an Error when the sidecar exits with an error. This is only apparent when using kubectl to get the pods of a TaskRun, not when describing the Pod using kubectl describe pod ... nor when looking at the TaskRun, but can be quite confusing.

[Tekton pipeline readme](https://github.com/tektoncd/pipeline/blob/master/docs/developers/README.md#handling-of-injected-sidecars) has documented this limitation, and the [tracking issue](https://github.com/tektoncd/pipeline/issues/1347). It has no function impacts.

<a name="store-secrets-to-pull-images" />

### Store the *secrets* which need to pull images

If your tetkon pipeline needs `secrets` to pull images from the private repository, currently the kfp-tekton compiler will generate a `ServiceAccount` and store the `secrets` in the `ServiceAccount`. But this approach is not ideal way to solve this problem, because it adds extra work to the server to maintain the `ServiceAccount`, the better approach will be using `podTemplate` to store the image pull `secrets`. The Tekton pipeline doesn't support this feature yet. Here is the [tracking issue](https://github.com/tektoncd/pipeline/issues/2339) in the tekton pipeline.

