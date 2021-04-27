# Custom Task: Pipeline Loops

`Pipeline Loops` implementation is based on [Tekton Custom Task](https://github.com/tektoncd/community/blob/master/teps/0002-custom-tasks.md) and the template of [Knative sample controller](https://github.com/knative-sandbox/sample-controller)

# Goal
`Pipeline Loops` is trying to provide pipeline level loops to handle `withItems` loop in `Pipeline`(Tekton).

# Installation

## Option One: Install using KO

- Install and configure [KO](https://github.com/google/ko)
- Install pipelineloop controller
  `ko apply -f config/`
  
## Option Two: Install by building your own Docker images

- Modify `Makefile` by changing `registry-name` to your Docker hub name

- Run `make images` to build the docker image of yours.

- Modify `config/500-webhook.yaml` and `config/500-controller.yaml` Change the image name to your docker image, e.g.:
```
- name: webhook
  image: fenglixa/pipelineloop-webhook:v0.0.1
```
```
- name: tekton-pipelineloop-controller
  image: fenglixa/pipelineloop-controller:v0.0.1
```

- Install pipelineloop controller `kubectl apply -f config/`


# Verification
- check controller and the webhook. `kubectl get po -n tekton-pipelines`
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

# End to end example
- Install Tekton version >= v0.19
- Edit feature-flags configmap, ensure "data.enable-custom-tasks" is "true":
`kubectl edit cm feature-flags -n tekton-pipelines`

- Run the E2E example: `kubectl apply -f examples/loop-example-basic.yaml`
