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

# Parallelism
`Parallelism` is to define the number of pipelineruns can be created at the same time. It must be eqaul to or bigger than 1. If it's not set then the default value will be 1. If it's bigger than the total iterations in the loop then the number will be total iterations.

# Break
It's common to break the loop when some condition met, like what we do in program language. This can be done by specify the task name with the `last-loop-task` label. When the task speicfied the the label is skipped during pipelineloop iteration, the loop will be marked `Succeeded` with a `pass` condition, and no more iteration will be started. The common use case is to check the condition in the task with `when` expression so that to decide whether to break the loop or just continue.

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

- Run the E2E example: 
  
  `kubectl apply -f examples/loop-example-basic.yaml`
  

- Tekton now supports custom task as embedded spec, it requires tekton version >= v0.25

  - Install Tekton version >= v0.25
  Or directly from source as,
  ```
    git clone https://github.com/tektoncd/pipeline.git
    cd pipeline
    make apply
  ```

  - To use the `taskSpec` example as below

    e.g.
    
    ```yaml
    apiVersion: tekton.dev/v1beta1
    kind: PipelineRun
    metadata:
      name: pr-loop-example
    spec:
      pipelineSpec:
        tasks:
          - name: first-task
            taskSpec:
              steps:
                - name: echo
                  image: ubuntu
                  imagePullPolicy: IfNotPresent
                  script: |
                    #!/usr/bin/env bash
                    echo "I am the first task before the loop task"
          - name: loop-task
            runAfter:
              - first-task
            params:
              - name: message
                value:
                  - I am the first one
                  - I am the second one
                  - I am the third one
            taskSpec:
              apiVersion: custom.tekton.dev/v1alpha1
              kind: PipelineLoop
              spec: # Following is the embedded spec.
                iterateParam: message
                pipelineSpec:
                  params:
                    - name: message
                      type: string
                  tasks:
                    - name: echo-loop-task
                      params:
                        - name: message
                          value: $(params.message)
                      taskSpec:
                        params:
                          - name: message
                            type: string
                        steps:
                          - name: echo
                            image: ubuntu
                            imagePullPolicy: IfNotPresent
                            script: |
                              #!/usr/bin/env bash
                              echo "$(params.message)"
          - name: last-task
            runAfter:
              - loop-task
            taskSpec:
              steps:
                - name: echo
                  image: ubuntu
                  imagePullPolicy: IfNotPresent
                  script: |
                    #!/usr/bin/env bash
                    echo "I am the last task after the loop task"
    ```

  - To run the above example:

    `kubectl apply -f examples/loop-example-basic_taskspec.yaml`

# Validation

A validation CLI validates any Pipeline/PipelineRun/Run with embedded
PipelineLoop Spec or any PipelineLoop custom task definition.

To build it from source, use the make tool as follows:


1. For both Mac and linux, invoke make with a default target. 

``` 
export BIN_DIR="bin"
make cli
```

Output:
```
mkdir -p bin
go build -o=bin/pipelineloop-cli ./cmd/cli
```

For linux specific build:
```
export BIN_DIR="bin"
make build-linux
```

2. Above command will generate the output in `bin` dir. Use as follows:
```
bin/pipelineloop-cli -f examples/loop-example-basic_taskspec.yaml 
```

Output:
```

Congratulations, all checks passed !!
```
