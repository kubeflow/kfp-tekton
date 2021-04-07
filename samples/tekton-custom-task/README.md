# Using Tekton Custom Task on KFP

This example shows how to define any Tekton Custom Task on KFP-Tekton and handle the custom task inputs and outputs.

## prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)

## Instructions
1. Install the Condtion custom task controller for computing runtime conditions for this example. Make sure to setup GOPATH and [ko](https://github.com/google/ko) before running the commands below.
   ```shell
   git clone https://github.com/tektoncd/experimental/
   cd experimental/cel
   ko apply -f config/
   ```

2. Compile the flip-coin pipeline using the compiler inside the python code. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `tekton-custom-task.yaml`.
    ```
    # Compile the python code
    python tekton-custom-task.py
    ```

Then, upload the `tekton-custom-task.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.


## Troubleshooting
Make sure the Tekton deployment enabled custom task as instructed in the KFP-Tekton deployment.
```shell
kubectl patch cm feature-flags -n tekton-pipelines \
     -p '{"data":{"disable-home-env-overwrite":"true","disable-working-directory-overwrite":"true", "enable-custom-tasks": "true"}}'
```
