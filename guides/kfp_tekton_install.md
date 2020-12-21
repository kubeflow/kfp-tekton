# Installing Kubeflow Pipelines with Tekton

## Table of Contents

- [Installation Targets and Prequisites](#installation-targets-and-prequisites)
  * [IBM Cloud Kubernetes Service (IKS)](#ibm-cloud-kubernetes-service-iks)
  * [OpenShift](#openshift)
  * [Other Cloud Providers or On-Prem Kubernetes Deployment](#other-cloud-providers-or-on-prem-kubernetes-deployment)
- [Standalone Kubeflow Pipelines with Tekton Backend Deployment](#standalone-kubeflow-pipelines-with-tekton-backend-deployment)
- [Kubeflow installation including Kubeflow Pipelines with Tekton Backend](#kubeflow-installation-including-kubeflow-pipelines-with-tekton-backend)
- [Troubleshooting](#troubleshooting)

## Installation Targets and Prequisites

A Kubernetes cluster `v1.16` that has least 8 vCPU and 16 GB memory.

### IBM Cloud Kubernetes Service (IKS)

   1. [Create an IBM Cloud cluster](https://www.kubeflow.org/docs/ibm/create-cluster/) or if you have an existing cluster, please follow the [initial setup for an existing cluster](https://github.com/kubeflow/website/blob/master/content/en/docs/ibm/create-cluster.md#connecting-to-an-existing-cluster)
   2. **Important**: Configure the IKS cluster with [IBM Cloud Block Storage Setup](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow/#ibm-cloud-block-storage-setup)

### OpenShift

   Follow the instructions at [Deploy Kubeflow Pipelines with Tekton backend on OpenShift Container Platform](./kfp-tekton-openshift.md). Depending on your situation, you can choose between the two approaches:
   1. Leverage OpenShift Pipelines (built on Tekton)
   2. Install Tekton as part of deployment

### Other Cloud Providers or On-Prem Kubernetes Deployment
   Visit [Kubeflow Cloud Installation](https://www.kubeflow.org/docs/started/cloud/) for setting up the preferred environment to deploy Kubeflow.

## Standalone Kubeflow Pipelines with Tekton Backend Deployment
To install the standalone Kubeflow Pipelines with Tekton, run the following steps:
1. Install [Tekton v0.16.3](https://github.com/tektoncd/pipeline/releases/tag/v0.16.3)

2. Install Kubeflow Pipelines with Tekton backend (kfp-tekton) v0.5.0 [custom resource definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)(CRDs).
   > Note: You can ignore the error `no matches for kind "Application" in version "app.k8s.io/v1beta1"` since it's a warning saying `application` CRD is not yet ready.
    ```shell
    kubectl apply --selector kubeflow/crd-install=true -f install/v0.5.0/kfp-tekton.yaml
    ```

3. Install Kubeflow Pipelines with Tekton backend (kfp-tekton) v0.5.0 deployment
    ```shell
    kubectl apply -f install/v0.5.0/kfp-tekton.yaml
    ```

4. Then, if you want to expose the Kubeflow Pipelines endpoint outside the cluster, run the following commands:
    ```shell
    kubectl patch svc ml-pipeline-ui -n kubeflow -p '{"spec": {"type": "LoadBalancer"}}'
    ```

    To get the Kubeflow Pipelines UI public endpoint using command line, run:
    ```shell
    kubectl get svc ml-pipeline-ui -n kubeflow -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
    ```

## Kubeflow installation including Kubeflow Pipelines with Tekton Backend

**Important: Please complete the [prequisites](#installation-targets-and-prequisites) before proceeding with the following instructions.**

1. Follow the [Kubeflow install instructions](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow/#kubeflow-installation) to install whole Kubeflow stack with the kfp-tekton installation. Kubeflow v1.2.0 uses Tekton v0.14.0 and kfp-tekton v0.4.0.

1. Visit [KFP Tekton User Guide](/guides/kfp-user-guide) and start learning how to use Kubeflow pipeline.

1. Visit [KFP Tekton Admin Guide](/guides/kfp-admin-guide.md) for how to configure kfp-tekton with different settings.


## Troubleshooting
 - (For IBM Cloud IKS users) If you accidentally deployed Kubeflow with IBM Cloud File Storage, run the below commands to remove the existing pvc. The below commands are for removing resources in multi-user, so you can ignore any missing pvc or rollout error if you are doing this for single user.
    ```shell
    kubectl delete pvc -n kubeflow katib-mysql metadata-mysql minio-pv-claim minio-pvc mysql-pv-claim
    kubectl delete pvc -n istio-system authservice-pvc
    kubectl rollout restart -n kubeflow deploy/mysql deploy/minio deploy/katib-mysql deploy/metadata-db
    kubectl rollout restart -n istio-system statefulset/authservice
    ```

    Then, redo the [Kubeflow install](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow/#kubeflow-installation) section to redeploy Kubeflow with the [block storageclass](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow/#ibm-cloud-block-storage-setup).

- If you redeploy Kubeflow and some components are not showing up, it was due to the [dynamic created webhook issue](https://github.com/kubeflow/manifests/issues/1379). This issue will be [fixed](https://github.com/kubeflow/pipelines/pull/4429) in the next release of KFP.
    ```shell
    kubectl delete MutatingWebhookConfiguration cache-webhook-kubeflow katib-mutating-webhook-config
    ```
