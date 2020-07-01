# The flip-coin pipeline

The [flip-coin pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/core/condition/condition.py)
is used to demonstrate the use of conditions.

* [Installing Tekton Pipelines on Kubernetes](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#installing-tekton-pipelines-on-kubernetes) 

* [Installing Tekton Pipelines on OpenShift](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#installing-tekton-pipelines-on-openshift)

There are two ways to generate `Tekton` YAML. One is using `dsl-compile-tekton` command line tool, another way is using `TektonCompiler` inside your pipeline script. Both will generate the same `Tekton` YAML file. Then either run the YAML to your Kubeflow cluster by using the Kubeflow pipeline UI or use the SDK command line.

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

Next, upload the `condition.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.
