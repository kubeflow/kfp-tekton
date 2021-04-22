# The flip-coin pipeline

This sample is a demostration on how to use user defined custom task and custom CEL task to reproduce the [flip-coin pipeline](../flip-coin/condition.py)
on different ways to define conditions and task dependencies.

## prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)

## Instructions
1. Install the Condtion custom task controller for computing runtime conditions. Make sure to setup GOPATH and [ko](https://github.com/google/ko) before running the commands below.
   ```shell
   git clone https://github.com/tektoncd/experimental/
   cd experimental/cel
   ko apply -f config/
   ```
2. Compile the flip-coin pipeline using the compiler inside the python code. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `condition.yaml`.
    ```shell
    # Compile the python code
    python condition.py
    ```

Then, upload the `condition.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.
