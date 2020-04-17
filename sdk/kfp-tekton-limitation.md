
## kfp-tekton limitation 
- Please be aware that defined Affinity, Node Selector, and Tolerations are applied to all the tasks in the same pipeline because there's only one podTemplate allowed in each pipeline.

- When you add test cases to compiler_tests, the output of pipeline/pipelinerun yaml may has uncertain values or orders, then you can define a lambda function as normalize_compiler_output_function to pass the testing.

- When you run kfp-tekton pipeline with sidecars, you may notice a completed pod and a sidercar pod with Error. This is a known limitation from tekton pipeline with sidecar output. [Tekton pipeline readme](https://github.com/tektoncd/pipeline/blob/master/docs/developers/README.md#handling-of-injected-sidecars) has documented this limitation, and the [tracking issue](https://github.com/tektoncd/pipeline/issues/1347). It has no function impacts.

- If your tetkon pipeline needs secrets to pull images from the private repository, currently the kfp-tekton compiler will generate a ServiceAccount and store the secrets in the ServiceAccount. But this approach is not idea way to solve this problem, because it adds extra work to the server to maintain the ServiceAccount, the better approach will be using podTemplate to store the image pull secret. The Tekton pipeline doesn't support this feature yet. Here is the [tracking issue](https://github.com/tektoncd/pipeline/issues/2339) in the tekton pipeline. 
