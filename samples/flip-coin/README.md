# The flip-coin pipeline

The [flip-coin pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/core/condition/condition.py)
is used to demonstrate the use of conditions.

## prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)

## Instructions
There are two ways to generate `Tekton` YAML. One is using `dsl-compile-tekton` command line tool, another way is using `TektonCompiler` inside your pipeline script. Both will generate the same `Tekton` YAML file.

* Using command line to generate YAML

    ```
    # Using kfp-tekton command line tool to generate `Tekton` YAML
    dsl-compile-tekton --py condition.py --output condition.yaml
    ```

* Using `TektonCompiler` to generate YAML
    ```
    # Compile the python code
    python condition.py
    ```

Then, upload the `condition.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.
