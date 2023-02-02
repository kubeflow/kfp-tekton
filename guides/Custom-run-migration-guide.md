# CustomRun migration guide - Applicable to tetkton v0.43.x and greater.

## CustomTask beta

[Official TEP with full details](https://github.com/tektoncd/community/blob/main/teps/0114-custom-tasks-beta.md)

1. Custom resource representing custom task feature in tektoncd is renamed from `Run` to `CustomRun` i.e. `v1aplha1.Run` is now `v1beta1.CustomRun`.
2. `v1alpha1.Run` is still supported by tekton and will be removed in future releases. Internally `v1alpha1.Run` is converted to its corresponding `v1beta1.CustomRun`.

An example of a custom task `Pipelineloop` being converted from `v1aplha1` API to `v1beta1` API.


### title {.tabset .tabset-fade}

#### tab v1aplha1.Run API
```yaml
---
apiVersion: custom.tekton.dev/v1alpha1
kind: PipelineLoop
metadata:
  name: simpletaskloop
spec:
  # Task to run (inline taskSpec also works)
  pipelineRef:
    name: demo-pipeline
  # Parameter that contains the values to iterate
  iterateParam: word
  # Timeout (defaults to global default timeout, usually 1h00m; use "0" for no timeout)
  timeout: 60s
---
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  name: simpletasklooprun
spec:
  params:
    - name: word
      value:
        - jump
        - land
        - roll
    - name: suffix
      value: ing
  ref:
    apiVersion: custom.tekton.dev/v1alpha1
    kind: PipelineLoop
    name: simpletaskloop
```
#### tab v1beta1.CustomRun API.

```yaml
---
apiVersion: custom.tekton.dev/v1alpha1
kind: PipelineLoop
metadata:
  name: simpletaskloop
spec:
  # Task to run (inline taskSpec also works)
  pipelineRef:
    name: demo-pipeline
  # Parameter that contains the values to iterate
  iterateParam: word
  # Timeout (defaults to global default timeout, usually 1h00m; use "0" for no timeout)
  timeout: 60s
---
apiVersion: tekton.dev/v1beta1
kind: CustomRun
metadata:
  name: simpletasklooprun
spec:
  params:
    - name: word
      value:
        - jump
        - land
        - roll
    - name: suffix
      value: ing
  customRef:
    apiVersion: custom.tekton.dev/v1alpha1
    kind: PipelineLoop
    name: simpletaskloop
```

## Custom task controller migration from `v1aplha1.Run` to `v1beta1.CustomRun`

1. Upgrade the tekton version to v0.43.x or higher in project `go.mod`, latest is better.
2. Perform `go mod tidy` and `go mod vendor` to upgrade the project to newer tekton version.
3. In the controller/reconciler code, start switching imports from `v1aplha1/run` to `v1beta1/customrun`. As an example this has been done for PipelineLoop controller in [this commit](https://github.com/kubeflow/kfp-tekton/commit/5512f1bfb21c2b4109b76132ba5ac586d3b995c4). 
4. References are changed from `Run.Spec.Ref` to `CustomRun.Spec.CustomRef`.
5. Pod Template field is now removed in `v1beta1.CustomRun`. In `PipelineLoop` controller we are storing pod templates as field inside the `PipelineLoop` custom resource.
6. We needed to convert the type(s) at one occasion thus wrote the following method:

```go

// FromRunStatus converts a *v1alpha1.RunStatus into a corresponding *v1beta1.CustomRunStatus
func FromRunStatus(orig *v1alpha1.RunStatus) *customRun.CustomRunStatus {
	crs := customRun.CustomRunStatus{
		Status: orig.Status,
		CustomRunStatusFields: customRun.CustomRunStatusFields{
			StartTime:      orig.StartTime,
			CompletionTime: orig.CompletionTime,
			ExtraFields:    orig.ExtraFields,
		},
	}

	for _, origRes := range orig.Results {
		crs.Results = append(crs.Results, customRun.CustomRunResult{
			Name:  origRes.Name,
			Value: origRes.Value,
		})
	}

	for _, origRetryStatus := range orig.RetriesStatus {
		crs.RetriesStatus = append(crs.RetriesStatus, customRun.FromRunStatus(origRetryStatus))
	}

	return &crs
}

```
