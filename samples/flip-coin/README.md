# The flip-coin pipeline

The [flip-coin pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/core/condition/condition.py)
is used to demonstrate the use of conditions.

* [Installing Tekton Pipelines on Kubernetes](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#installing-tekton-pipelines-on-kubernetes) 

* [Installing Tekton Pipelines on OpenShift](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#installing-tekton-pipelines-on-openshift)

There are two ways to generate `Tekton` YAML. One is using `dsl-compile-tekton` command line tool, another way is using `TektonCompiler` inside your pipeline script. Both will generate the same `Tekton` YAML file. Then apply the YAML to your cluster by using the `tkn` command.

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

```
# Install tasks, conditions and pipeline
kubectl apply -f condition.yaml

# Check the pipeline is running in your cluster, should see something like `conditional-execution-pipeline` is running
kubectl get pipeline

# Start the pipeline
tkn pipeline start conditional-execution-pipeline --showlog
```

If it runs successfully, you should see something like this:
```
Pipelinerun started: conditional-execution-pipeline-run-vjkhz
Waiting for logs to be available...
[flip-coin : flip-coin] tails

[generate-random-number-2 : generate-random-number-2] 17

[print-3 : print-3] tails and 17
[print-3 : print-3]  > 15!
```

