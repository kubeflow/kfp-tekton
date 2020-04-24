# The flip-coin pipeline

The [flip-coin pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/core/condition/condition.py)
is used to demonstrate the use of conditions.

* [Installing Tekton Pipelines on Kubernetes](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#installing-tekton-pipelines-on-kubernetes) 

* [Installing Tekton Pipelines on OpenShift](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#installing-tekton-pipelines-on-openshift)

* [Configuring artifact storage](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#configuring-artifact-storage): When using S3 bucket, make sure to update your bucket name.

There are two ways to generate `Tekton` YAML. One is using `DSL-COMPILE-TEKTON` command line tool, another way is using `TektonCompiler` inside your pipeline script. Both will generate a same `Tekton` YAML file. Then apply the YAML to your cluster by using `tkn` command.

* Using command line to generate YAML

    ```
    # Using kfp-tekton commnad line tool to generate `Tekton` YAML
    dsl-compile-tekton --py condition.py --output condition.yaml
    ```

* Using `TektonCompiler` to generate YAML
    ```
    # Replace the kfp compile code in the condition.py's `main`: 
    # From
    #   `kfp.compiler.Compiler().compile(flipcoin_pipeline, __file__ + '.yaml')`
    # To
    #   `from kfp_tekton.compiler import TektonCompiler`
    #   `TektonCompiler().compile(flipcoin_pipeline, __file__.replace('.py', '.yaml'))`
    python condition.py
    ```

```
# Install tasks, conditions and pipeline
kubectl apply -f condition.yaml

# Check the pipeline is running in your cluster, should see somethig like `conditional-execution-pipeline` is running
kubectl get pipeline

# Start the pipeline
tkn pipeline start conditional-execution-pipeline --showlog
```

If it is success, you will see something like this:
```
Pipelinerun started: conditional-execution-pipeline-run-vjkhz
Waiting for logs to be available...
[flip-coin : flip-coin] tails

[generate-random-number-2 : generate-random-number-2] 17

[print-3 : print-3] tails and 17
[print-3 : print-3]  > 15!
```

