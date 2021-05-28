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

