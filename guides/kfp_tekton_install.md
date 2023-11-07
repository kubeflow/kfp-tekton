# Installing Kubeflow Pipelines with Tekton

## Table of Contents

- [Installation Targets and Prerequisites](#installation-targets-and-prerequisites)
  * [IBM Cloud Kubernetes Service (IKS)](#ibm-cloud-kubernetes-service-iks)
  * [OpenShift](#openshift)
  * [Other Cloud Providers or On-Prem Kubernetes Deployment](#other-cloud-providers-or-on-prem-kubernetes-deployment)
  * [KIND deployment](#kind-deployment)
  * [Compatibility Map](#compatibility-map)
- [Standalone Kubeflow Pipelines V1 with Tekton Backend Deployment](#standalone-kubeflow-pipelines-v1-with-tekton-backend-deployment)
- [Standalone Kubeflow Pipelines V2 with Tekton Backend Deployment](#standalone-kubeflow-pipelines-v2-with-tekton-backend-deployment)
- [Standalone Kubeflow Pipelines with Openshift Pipelines Backend Deployment](#standalone-kubeflow-pipelines-with-openshift-pipelines-backend-deployment)
- [Kubeflow installation including Kubeflow Pipelines with Tekton Backend](#kubeflow-installation-including-kubeflow-pipelines-with-tekton-backend)
- [Upgrade to Multi-User KFP-Tekton on Kubeflow](#upgrade-to-multi-user-kfp-tekton-on-kubeflow)
- [Troubleshooting](#troubleshooting)

## Installation Targets and Prerequisites

A Kubernetes cluster `v1.25` that has least 8 vCPU and 16 GB memory.

### IBM Cloud Kubernetes Service (IKS)

   1. [Create an IBM Cloud cluster](https://www.kubeflow.org/docs/ibm/create-cluster/) or if you have an existing cluster, please follow the [initial setup for an existing cluster](https://master.kubeflow.org/docs/distributions/ibm/create-cluster/#connecting-to-an-existing-cluster)
   2. **Important**: Configure the IKS cluster with [IBM Cloud Group ID Storage Setup](https://www.kubeflow.org/docs/distributions/ibm/deploy/install-kubeflow-on-iks/#storage-setup-for-a-classic-ibm-cloud-kubernetes-cluster)

### OpenShift

   Depending on your situation, you can choose between the two approaches to set up the pipeline engine on Openshift:
   1. Using [OpenShift Pipelines](https://docs.openshift.com/container-platform/4.12/cicd/pipelines/installing-pipelines.html) (built on Tekton), follow the [Standalone Kubeflow Pipelines with Openshift Pipelines Backend Deployment](#standalone-kubeflow-pipelines-with-openshift-pipelines-backend-deployment)
   2. Using [Tekton on Openshift](https://github.com/tektoncd/pipeline/blob/v0.47.1/docs/install.md#installing-tekton-pipelines-on-openshift), follow the [Standalone Kubeflow Pipelines with Tekton Backend Deployment](#standalone-kubeflow-pipelines-with-tekton-backend-deployment) to install the Kubeflow Pipeline Stack. Note the current Tekton Open Source deployment for [Openshift doesn't work out of the box](https://github.com/tektoncd/pipeline/issues/3452), so we strongly recommend to deploy with Opneshift Pipelines (see above) if you want to run Kubeflow Pipelines on Openshift.

### Other Cloud Providers or On-Prem Kubernetes Deployment

   Visit [Kubeflow Installation](https://www.kubeflow.org/docs/started/) for setting up the preferred environment to deploy Kubeflow.

### KIND deployment

   If you want to deploy locally on [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/#installation), you can run the kubectl Kustomization command below
   ```bash
   kubectl apply -k https://github.com/kubeflow/kfp-tekton//manifests/kustomize/env/platform-agnostic-kind\?ref\=v1.8.1
   ```
   
### Compatibility Map

Each new KFP-Tekton version is based on the long-term support of the Tekton Pipeline version and the major release of the Openshift pipeline version. Below is the list of compatible KFP-Tekton version to the Tekton/Openshift pipelines version.
   
| KFP-Tekton Version    | Tekton Pipeline Version | OpenShift Pipelines Version | Tekton Core API Version | KFP GRPC Gateway Version |
| -------- | ------- | ------- | ------- | ------- |
| 1.5.x    | 0.41.x  | 1.9     | V1beta1 | 1.16.0  |
| 1.6.x    | 0.44.x  | 1.10    | V1beta1 | 1.16.0  |
| 1.7.x    | 0.47.x  | 1.11    | V1beta1 | 1.16.0  |
| 1.8.x    | 0.50.x  | 1.12    | V1      | 2.11.3  |
| 2.0.x    | 0.47.x  | 1.11    | V1      | 1.16.0  |

## Standalone Kubeflow Pipelines V1 with Tekton Backend Deployment

To install the standalone Kubeflow Pipelines V1 with Tekton , run the following steps:

1. Install [Tekton v0.50.2](https://github.com/tektoncd/pipeline/blob/v0.50.2/docs/install.md#installing-tekton-pipelines-on-kubernetes) if you don't have Tekton pipelines on the cluster. Please be aware that Tekton custom task, loop, and recursion will not work if Tekton pipelines version is not v0.50.2+.

   ```shell
   kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.50.1/release.yaml
   ```

2. Enable necessary Tekton configurations for kfp-tekton
   ```shell
   kubectl patch cm feature-flags -n tekton-pipelines \
         -p '{"data":{"running-in-environment-with-injected-sidecars": "false"}}'
   kubectl patch cm config-defaults -n tekton-pipelines \
         -p '{"data":{"default-timeout-minutes": "0"}}'
   ```

3. Install Kubeflow Pipelines with Tekton backend (`kfp-tekton`) `v1.8.1` [custom resource definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)(CRDs).
    ```shell
    kubectl apply --selector kubeflow/crd-install=true -f https://raw.githubusercontent.com/kubeflow/kfp-tekton/master/install/v1.8.1/kfp-tekton.yaml
    ```

4. Install Kubeflow Pipelines with Tekton backend (`kfp-tekton`) `v1.8.1` deployment
    ```shell
    kubectl apply -f https://raw.githubusercontent.com/kubeflow/kfp-tekton/master/install/v1.8.1/kfp-tekton.yaml
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
   curl -L https://raw.githubusercontent.com/kubeflow/kfp-tekton/master/install/v1.8.1/kfp-tekton.yaml | yq 'del(.spec.template.spec.containers[].securityContext.runAsUser, .spec.template.spec.containers[].securityContext.runAsGroup)' | oc apply -f -
   oc apply -k https://github.com/kubeflow/kfp-tekton//manifests/kustomize/third-party/openshift/standalone
   oc adm policy add-scc-to-user anyuid -z tekton-pipelines-controller
   oc adm policy add-scc-to-user anyuid -z tekton-pipelines-webhook
   ```

## Standalone Kubeflow Pipelines V2 with Tekton Backend Deployment

To install the standalone Kubeflow Pipelines V2 with Tekton, run the following steps:

1. Install Kubeflow Pipelines with Tekton backend (`kfp-tekton`) `v2.0.3` along with Tekton `v0.47.5`
   ```shell
   kubectl apply -k https://github.com/kubeflow/kfp-tekton//manifests/kustomize/env/platform-agnostic-tekton\?ref\=v2.0.3
   ```

2. Then, if you want to expose the Kubeflow Pipelines endpoint outside the cluster, run the following commands:
    ```shell
    kubectl patch svc ml-pipeline-ui -n kubeflow -p '{"spec": {"type": "LoadBalancer"}}'
    ```

    To get the Kubeflow Pipelines UI public endpoint using command line, run:
    ```shell
    kubectl get svc ml-pipeline-ui -n kubeflow -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
    ```

3. (GPU worker nodes only) If your Kubernetes cluster has a mixture of CPU and GPU worker nodes, it's recommended to disable the Tekton default affinity assistant so that Tekton won't schedule too many CPU workloads on the GPU nodes.
    ```shell
    kubectl patch cm feature-flags -n tekton-pipelines \
      -p '{"data":{"disable-affinity-assistant": "true"}}'
    ```

Now, please use the [KFP V2 Python SDK](https://pypi.org/project/kfp/) to compile KFP-Tekton V2 pipelines because we are sharing the same pipeline spec starting from KFP V2.0.0. 

```shell
pip install "kfp>=2.0" "kfp-kubernetes>=1.0.0"
```

## Standalone Kubeflow Pipelines with Openshift Pipelines Backend Deployment

To install the standalone Kubeflow Pipelines with Openshift Pipelines, run the following steps:

1. Install openshift pipelines (v1.12) from openshift operatorhub:

![openshift-pipelines](/images/openshift-pipelines.png)

2. Enable necessary Openshift pipelines configurations for kfp-tekton to enable high performance pipelines.
   ```shell
   oc patch cm feature-flags -n openshift-pipelines \
         -p '{"data":{"running-in-environment-with-injected-sidecars": "false"}}'
   oc patch cm config-defaults -n openshift-pipelines \
         -p '{"data":{"default-timeout-minutes": "0"}}'
   ```

3. Install Kubeflow Pipelines with Openshift pipelines backend (`kfp-tekton`) `v1.8.1` deployment
   ```shell
   oc apply -k https://github.com/kubeflow/kfp-tekton//manifests/kustomize/env/kfp-template-openshift-pipelines\?ref\=v1.8.1
   ```

4. Then, if you want to expose the Kubeflow Pipelines endpoint outside the cluster, run the following commands:
    ```shell
    kubectl patch svc ml-pipeline-ui -n kubeflow -p '{"spec": {"type": "LoadBalancer"}}'
    ```

    To get the Kubeflow Pipelines UI public endpoint using command line, run:
    ```shell
    kubectl get svc ml-pipeline-ui -n kubeflow -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
    ```

5. (GPU worker nodes only) If your Openshift cluster has a mixture of CPU and GPU worker nodes, it's recommended to disable the Openshift pipelines default affinity assistant so that Openshift pipelines won't schedule too many CPU workloads on the GPU nodes.
    ```shell
    oc patch cm feature-flags -n openshift-pipelines \
      -p '{"data":{"disable-affinity-assistant": "true"}}'
    ```

## Kubeflow installation including Kubeflow Pipelines with Tekton Backend

**Important: Please complete the [prerequisites](#installation-targets-and-prerequisites) before proceeding with the following instructions.**

1. Follow the [Kubeflow install instructions](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow-on-iks/#kubeflow-installation)
   to install the entire Kubeflow stack with `kfp-tekton`.
   Kubeflow `v1.8.0` uses Tekton `v0.47.3` and `kfp-tekton` `v2.0.3` or `v1.7.1`. <!-- TODO update-->

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
