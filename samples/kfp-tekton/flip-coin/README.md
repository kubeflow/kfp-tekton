# The flip-coin pipeline

The [flip-coin pipeline](https://github.com/kubeflow/pipelines/blob/master/samples/core/condition/condition.py)
is used to demonstrate the use of conditions. 

* Argo: the [compiled Argo version](https://github.com/kubeflow/kfp-tekton/blob/master/kfp-samples/conditionsflip-coin-kfp.yaml) was produced using the tkp DSL compiler
* Tekton: the [Tekton]() version was built to match the Argo one. This pipeline is over-engineered for what it does, but it demonstrate using `Condition`s in combination with `Task`s and `PipelineResource`s. To run this pipeline using `tkn`:

```
# Install tasks, conditions and pipeline
kubectl apply -f kfp-samples/condition/flip-coin-tekton.yaml

# Prepare the resources and apply them
kubectl apply -f kfp-sapmles/condition/flip-coin-tekton-resources.yaml

# Run the pipeline
tkn pipeline start flip-coin-condition-demo --resource=coin=coin-bucket --resource=random=random-bucket
```
