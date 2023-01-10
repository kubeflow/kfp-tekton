# Installing Kubeflow Pipelines with Tekton

## Table of Contents

- [Installation Targets and Prerequisites](#installation-targets-and-prerequisites)
  * [IBM Cloud Kubernetes Service (IKS)](#ibm-cloud-kubernetes-service-iks)
  * [OpenShift](#openshift)
  * [Other Cloud Providers or On-Prem Kubernetes Deployment](#other-cloud-providers-or-on-prem-kubernetes-deployment)
  * [Alternative KIND deployment](#alternative-kind-deployment)
- [Standalone Kubeflow Pipelines with Tekton Backend Deployment](#standalone-kubeflow-pipelines-with-tekton-backend-deployment)
- [Kubeflow installation including Kubeflow Pipelines with Tekton Backend](#kubeflow-installation-including-kubeflow-pipelines-with-tekton-backend)
- [Upgrade to Multi-User KFP-Tekton on Kubeflow](#upgrade-to-multi-user-kfp-tekton-on-kubeflow)
- [Troubleshooting](#troubleshooting)

## Installation Targets and Prerequisites

A Kubernetes cluster `v1.23` that has least 8 vCPU and 16 GB memory.

### IBM Cloud Kubernetes Service (IKS)

   1. [Create an IBM Cloud cluster](https://www.kubeflow.org/docs/ibm/create-cluster/) or if you have an existing cluster, please follow the [initial setup for an existing cluster](https://master.kubeflow.org/docs/distributions/ibm/create-cluster/#connecting-to-an-existing-cluster)
   2. **Important**: Configure the IKS cluster with [IBM Cloud Group ID Storage Setup](https://www.kubeflow.org/docs/distributions/ibm/deploy/install-kubeflow-on-iks/#storage-setup-for-a-classic-ibm-cloud-kubernetes-cluster)

### OpenShift

   Depending on your situation, you can choose between the two approaches to set up the pipeline engine on Openshift:
   1. Leverage [OpenShift Pipelines](https://docs.openshift.com/container-platform/4.9/cicd/pipelines/installing-pipelines.html) (built on Tekton)
   2. Install Tekton as part of deployment

   Once you decided your approach, follow the [Standalone Kubeflow Pipelines with Tekton Backend Deployment](#standalone-kubeflow-pipelines-with-tekton-backend-deployment) to install the Kubeflow Pipeline Stack.

### Other Cloud Providers or On-Prem Kubernetes Deployment

   Visit [Kubeflow Installation](https://www.kubeflow.org/docs/started/) for setting up the preferred environment to deploy Kubeflow.

### Alternative KIND deployment

   If you want to deploy locally, you can [deploy MLX on KIND](https://github.com/machine-learning-exchange/mlx/blob/main/docs/install-mlx-on-kind.md). MLX in build on top of kfp-tekton, so you will have Kubeflow Pipeline with Tekton installed after finish deploy MLX on KIND.

## Standalone Kubeflow Pipelines with Tekton Backend Deployment

To install the standalone Kubeflow Pipelines with Tekton, run the following steps:

1. Install [Tekton v0.41.0](https://github.com/tektoncd/pipeline/blob/v0.41.0/docs/install.md#installing-tekton-pipelines-on-kubernetes) if you don't have Tekton pipelines or OpenShift Pipelines on the cluster. Please be aware that Tekton custom task, loop, and recursion will not work if Tekton/Openshift pipelines version is not v0.28.0+.

   ```shell
   kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.41.0/release.yaml
   ```

2. Enable custom task controller and other feature flags for kfp-tekton
   ```shell
   kubectl patch cm feature-flags -n tekton-pipelines \
         -p '{"data":{"enable-custom-tasks": "true"}}'
   kubectl patch cm config-defaults -n tekton-pipelines \
         -p '{"data":{"default-timeout-minutes": "0"}}'
   ```

3. Install Kubeflow Pipelines with Tekton backend (`kfp-tekton`) `v1.5.0` [custom resource definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)(CRDs).
   > Note: You can ignore the error `no matches for kind "Application" in version "app.k8s.io/v1beta1"` since it's a warning saying `application` CRD is not yet ready.
    ```shell
    kubectl apply --selector kubeflow/crd-install=true -f install/v1.5.0/kfp-tekton.yaml
    ```

4. Install Kubeflow Pipelines with Tekton backend (`kfp-tekton`) `v1.5.0` deployment
    ```shell
    kubectl apply -f install/v1.5.0/kfp-tekton.yaml
    ```

5. Then, if you want to expose the Kubeflow Pipelines endpoint outside the cluster, run the following commands:
    ```shell
    kubectl patch svc ml-pipeline-ui -n kubeflow -p '{"spec": {"type": "LoadBalancer"}}'
    ```

    To get the Kubeflow Pipelines UI public endpoint using command line, run:
    ```shell
    kubectl get svc ml-pipeline-ui -n kubeflow -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
    ```

6. (GPU worker nodes only) If your Kubernetes cluster has a mixture of CPU and GPU worker nodes, it's recommended to disable the Tekton default affinity assistant so that Tekton won't schedule too many CPU workloads on the GPU nodes.
    ```shell
    kubectl patch cm feature-flags -n tekton-pipelines \
      -p '{"data":{"disable-affinity-assistant": "true"}}'
    ```

7. (OpenShift only) If you are running the standalone KFP-Tekton on OpenShift, apply the necessary security context constraint below
   ```shell
   oc apply -k manifests/kustomize/third-party/openshift/standalone
   ```

## Kubeflow installation including Kubeflow Pipelines with Tekton Backend

**Important: Please complete the [prerequisites](#installation-targets-and-prerequisites) before proceeding with the following instructions.**

1. Follow the [Kubeflow install instructions](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow-on-iks/#kubeflow-installation)
   to install the entire Kubeflow stack with `kfp-tekton`.
   Kubeflow `v1.6.0` uses Tekton `v0.31.4` and `kfp-tekton` `v1.2.1`. <!-- TODO update-->

2. Visit [KFP Tekton User Guide](/guides/kfp-user-guide) and start learning how to use Kubeflow pipeline.

3. Visit [KFP Tekton Admin Guide](/guides/kfp-admin-guide.md) for how to configure kfp-tekton with different settings.


## Upgrade to Multi-User KFP-Tekton on Kubeflow

1. Starting from Kubeflow 1.3 and beyond, both Kubeflow single and multi-user deployment use the multi-user mode of Kubeflow pipelines to support authentication. If you haven't installed Kubeflow, Follow the [Kubeflow install instructions](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow-on-iks/#kubeflow-installation) to install Kubeflow Pipelines with multi-user capabilities.

2. To upgrade to the Multi-User version of KFP-Tekton, custom task controllers, and core Tekton controller, please run

   ```shell
   kubectl apply -k manifests/kustomize/env/platform-agnostic-multi-user
   ```

   If you only want to upgrade the core KFP-Tekton (no custom task and Tekton upgrade), run

   ```shell
   kubectl apply -k manifests/kustomize/env/plain-multi-user
   ```


## Troubleshooting

 - (For IBM Cloud IKS users) If you accidentally deployed Kubeflow with IBM Cloud File Storage, run the below commands to remove the existing pvc. The below commands are for removing resources in multi-user, so you can ignore any missing pvc or rollout error if you are doing this for single user.
    ```shell
    kubectl delete pvc -n kubeflow katib-mysql metadata-mysql minio-pv-claim minio-pvc mysql-pv-claim
    kubectl delete pvc -n istio-system authservice-pvc
    kubectl rollout restart -n kubeflow deploy/mysql deploy/minio deploy/katib-mysql deploy/metadata-db
    kubectl rollout restart -n istio-system statefulset/authservice
    ```

    Then, redo the [Kubeflow install](https://www.kubeflow.org/docs/distributions/ibm/deploy/install-kubeflow-on-iks/#installation) section to redeploy Kubeflow with the appropriate storage setup. Either for a [Classic IBM Cloud Kubernetes cluster](https://www.kubeflow.org/docs/distributions/ibm/deploy/install-kubeflow-on-iks/#storage-setup-for-a-classic-ibm-cloud-kubernetes-cluster) or a [vpc-gen2 IBM Cloud Kubernetes cluster](https://www.kubeflow.org/docs/distributions/ibm/deploy/install-kubeflow-on-iks/#storage-setup-for-vpc-gen2-ibm-cloud-kubernetes-cluster).

- If you redeploy Kubeflow and some components are not showing up, it was due to the [dynamic created webhook issue](https://github.com/kubeflow/manifests/issues/1379). This issue will be [fixed](https://github.com/kubeflow/pipelines/pull/4429) in the next release of KFP.
    ```shell
    kubectl delete MutatingWebhookConfiguration cache-webhook-kubeflow katib-mutating-webhook-config
    ```
