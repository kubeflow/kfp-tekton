# Retrieve KFP metadata using Kubernetes downstream API

KFP-Tekton by default comes with a lot of the pipeline metadata that are avalible as part of the pipeline execution pod. However, in order to get those metadata inside the pipeline, users need to leveage the [Kubernetes downstream API](https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/) feature. This pipeline is a basic example using Kubernete downstream API to get KFP run_name and run_id as environment variables inside the pipeline.

## prerequisites
- Install [KFP Tekton prerequisites](/samples/README.md)

## Instructions
* Compile the flip-coin pipeline using the compiler inside the python code. The kfp-tekton SDK will produce a Tekton pipeline yaml definition in the same directory called `k8s-downstream-api.yaml`.
    ```
    # Compile the python code
    python k8s-downstream-api.py
    ```

Then, upload the `k8s-downstream-api.yaml` file to the Kubeflow pipeline dashboard with Tekton Backend to run this pipeline.
