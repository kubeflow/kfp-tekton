# Custom Task: Pipeline Loops

`Pipeline Loops` is implements based on [Tekton Custom Task](https://github.com/tektoncd/community/blob/master/teps/0002-custom-tasks.md) and the template of [Knative sample controller](https://github.com/knative-sandbox/sample-controller)

# Goal
`Pipeline Loops` is trying to provide a pipeline level loops to handle withItems loop in `Pipeline`(Tekton).

# Take a try
- Generate the code of client
  `go mod vendor; go mod tidy; bash hack/update-codegen.sh`
- Install&config [KO](https://github.com/google/ko)
- Install Exception Handler
  `ko apply -f config/`
- Verification
  `kubectl get po -n tekton-pipelines`
   ```
    ...
    tekton-pipelineloop-controller-db4c7dddb-vrlsd                        1/1     Running     0          6h24m
    tekton-pipelineloop-webhook-7bb98ddc98-qqkv6                          1/1     Running     0          6h17m
   ```
- Try cases to loop pipelines
  - `kubectl apply -f examples/pipelinespec-with-run-arrary-value.yaml`
  - `kubectl apply -f examples/pipelinespec-with-run-string-value.yaml`
  - `kubectl apply -f examples/pipelinespec-with-run-iterate-numeric.yaml`
  - `kubectl apply -f examples/pipelinespec-with-run-condition.yaml`

# E2E case
- Install Tekton version >= v0.19
- Edit feature-flags configmap, ensure "data.enable-custom-tasks" is "true":
`kubectl edit cm feature-flags -n tekton-pipelines`

- Run the E2E cases: `kubectl apply -f examples/loop-example-basic.yaml`