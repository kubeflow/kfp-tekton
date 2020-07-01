# The flip-coin pipeline

The [flip-coin pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/core/condition/condition.py)
is used to demonstrate the use of conditions.

## prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)

## Instructions
* Compile the flip-coin pipeline using the compiler inside the python code.
    ```
    # Compile the python code
    python condition.py
    ```

Then, upload the `condition.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.
