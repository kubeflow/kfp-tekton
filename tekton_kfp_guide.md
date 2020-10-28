# Installing Kubeflow Pipelines with Tekton

## Table of Contents

- [Installation Targets and Prequisites](#installation-targets-and-prequisites)
  * [IBM Cloud Kubernetes Service (IKS)](#ibm-cloud-kubernetes-service-iks)
  * [OpenShift](#openshift)
  * [Other Cloud Providers or On-Prem Kubernetes Deployment](#other-cloud-providers-or-on-prem-kubernetes-deployment)
- [Minimum Kubeflow Pipelines with Tekton Backend Deployment](#minimum-kubeflow-pipelines-with-tekton-backend-deployment)
- [Kubeflow installation including Kubeflow Pipelines with Tekton backend](#kubeflow-installation-including-kubeflow-pipelines-with-tekton-backend)
  * [Single user](#single-user)
  * [Multi-user, auth-enabled](#multi-user-auth-enabled)
- [Verify installation](#verify-installation)
- [Understanding the Kubeflow deployment process](#understanding-the-kubeflow-deployment-process)
  * [App layout](#app-layout)
- [Troubleshooting](#troubleshooting)

## Installation Targets and Prequisites

A Kubernetes cluster `v1.16` that has least 8 vCPU and 16 GB memory.

### IBM Cloud Kubernetes Service (IKS)

   1. [Create an IBM Cloud cluster](https://www.kubeflow.org/docs/ibm/create-cluster/) or if you have an existing cluster, please follow the [initial setup for an existing cluster](https://www.kubeflow.org/docs/ibm/existing-cluster/)
   2. **Important**: Configure the IKS cluster with [IBM Cloud Block Storage Setup](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow/#ibm-cloud-block-storage-setup)

### OpenShift

   Follow the instructions at [Deploy Kubeflow Pipelines with Tekton backend on OpenShift Container Platform](https://github.com/IBM/KubeflowDojo/tree/master/OpenShift/manifests). Depending on your situation, you can choose between the two approaches:
   1. Leverage OpenShift Pipelines (built on Tekton)
   2. Install Tekton as part of deployment

### Other Cloud Providers or On-Prem Kubernetes Deployment
   Visit [Kubeflow Cloud Installation](https://www.kubeflow.org/docs/started/cloud/) for setting up the preferred environment to deploy Kubeflow.

## Minimum Kubeflow Pipelines with Tekton Backend Deployment
To install the very minimum Kubeflow pipelines deployment without any other kubeflow functionality, run the following steps:
1. Install [Tekton v0.14.3](https://github.com/tektoncd/pipeline/releases/tag/v0.14.3)

2. Install KFP Tekton v0.4.0 release
    ```shell
    kubectl apply -f install/v0.4.0/kfp-tekton.yaml
    ```

3. Then, if you want to expose the Kubeflow pipelines to the public web, run the following commands:
    ```shell
    kubectl patch svc ml-pipeline-ui -n kubeflow -p '{"spec": {"type": "LoadBalancer"}}'
    ```

    To get the Kubeflow pipelines UI public endpoint using command line, run:
    ```shell
    kubectl get svc ml-pipeline-ui -n kubeflow -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
    ```

## Kubeflow installation including Kubeflow Pipelines with Tekton backend

**Important: Please complete the [prequisites](#installation-targets-and-prequisites) before proceeding with the following instructions.**

Run the following commands to set up and deploy Kubeflow with KFP-Tekton. To understand more about the Kubeflow deployment mechanism, please read [here](#understanding-the-kubeflow-deployment-process).

1. Download the kfctl v1.1.0 release from the
  [Kubeflow releases
  page](https://github.com/kubeflow/kfctl/releases/tag/v1.1.0).

1. Unpack the tarball
      ```
      tar -xvf kfctl_v1.1.0_<platform>.tar.gz
      ```
1. Make kfctl binary easier to use (optional). If you don’t add the binary to your path, you must use the full path to the kfctl binary each time you run it.
      ```
      export PATH=$PATH:<path to where kfctl was unpacked>
      ```

Choose either [**single user**](#single-user) or [**multi-tenant**](#multi-user-auth-enabled) section based on your usage.

### Single user
Run the following commands to set up and deploy Kubeflow for a single user without any authentication.
```
# Set KF_NAME to the name of your Kubeflow deployment. This also becomes the
# name of the directory containing your configuration.
# For example, your deployment name can be 'my-kubeflow' or 'kf-test'.
export KF_NAME=<your choice of name for the Kubeflow deployment>

# Set the path to the base directory where you want to store one or more
# Kubeflow deployments. For example, /opt/.
# Then set the Kubeflow application directory for this deployment.
export BASE_DIR=<path to a base directory>
export KF_DIR=${BASE_DIR}/${KF_NAME}

# Set the configuration file to use, such as the file specified below:
export CONFIG_URI="https://raw.githubusercontent.com/IBM/KubeflowDojo/master/manifests/kfctl_k8s_single_user.yaml"

# Generate and deploy Kubeflow:
mkdir -p ${KF_DIR}
cd ${KF_DIR}
kfctl apply -V -f ${CONFIG_URI}
```

* **${KF_NAME}** - The name of your Kubeflow deployment.
  If you want a custom deployment name, specify that name here.
  For example,  `my-kubeflow` or `kf-test`.
  The value of KF_NAME must consist of lower case alphanumeric characters or
  '-', and must start and end with an alphanumeric character.
  The value of this variable cannot be greater than 25 characters. It must
  contain just a name, not a directory path.
  This value also becomes the name of the directory where your Kubeflow
  configurations are stored, that is, the Kubeflow application directory.

* **${KF_DIR}** - The full path to your Kubeflow application directory.

### Multi-user, auth-enabled
Run the following commands to deploy multi-user auth-enabled Kubeflow with GitHub OAuth as the Dex authentication provider. To support multi-users with authentication enabled, this guide uses [Dex](https://github.com/dexidp/dex) with [GitHub OAuth](https://docs.github.com/developers/apps/building-oauth-apps).

**Before continuing, refer to the guide [Creating an OAuth App](https://docs.github.com/developers/apps/creating-an-oauth-app) for steps to create an OAuth app on GitHub.com.**

The scenario is a GitHub organization owner can authorize its organization members to access a deployed kubeflow. A member of this GitHub organization will be redirected to a page to grant access to the GitHub profile by Kubeflow.

1. Create a new OAuth app in GitHub. Use following setting to register the application:
    * Homepage URL: `http://<node_pubilc_ip>:31380/`
    * Authorization callback URL: `http://<node_pubilc_ip>:31380/dex/callback`
1. Once the application is registered, copy and save the `Client ID` and `Client Secret` for use later.
1. Setup environment variables:
    ```
    export KF_NAME=<your choice of name for the Kubeflow deployment>

    # Set the path to the base directory where you want to store one or more
    # Kubeflow deployments. For example, /opt/.
    export BASE_DIR=<path to a base directory>

    # Then set the Kubeflow application directory for this deployment.
    export KF_DIR=${BASE_DIR}/${KF_NAME}
    ```
1. Setup configuration files:
    ```
    export CONFIG_URI="https://raw.githubusercontent.com/IBM/KubeflowDojo/master/manifests/kfctl_dex_multi_user_tekton_V1.1.0.yaml"
    # Generate and deploy Kubeflow:
    mkdir -p ${KF_DIR}
    cd ${KF_DIR}
    ```
1. Deploy Kubeflow:
    ```
    kfctl apply -V -f ${CONFIG_URI}
    ```
1. Wait until the deployment finishes successfully. e.g., all pods are in `Running` state when running `kubectl get pod -n kubeflow`.
1. Update the configmap `dex` in namespace `auth` with credentials from the first step.
    - Get current resource file of current configmap `dex`:
    `kubectl get configmap dex -n auth -o yaml > dex-cm.yaml`
    - Replace the following values in the `dex-cm.yaml`.
      - **clientID**: Client ID of the Github OAuth App
      - **clientSecret**: Client Secret of the Github OAuth App
      - **orgs[0].name**: Any [Github organization](https://docs.github.com/en/github/setting-up-and-managing-organizations-and-teams/about-your-organization-dashboard) that the OAuth App has member read access.
      - (Optional) **hostName**: GitHub Enterprise Hostname. Leave it empty when using github.com.

    The `dex-cm.yaml` file looks like following:
    ```YAML
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: dex
      namespace: auth
    data:
      config.yaml: |-
        issuer: http://dex.auth.svc.cluster.local:5556/dex
        storage:
          type: kubernetes
          config:
            inCluster: true
        web:
          http: 0.0.0.0:5556
        logger:
          level: "debug"
          format: text
        connectors:
          - type: github
            # Required field for connector id.
            id: github
            # Required field for connector name.
            name: GitHub
            config:
              # Fill in client ID, client Secret as string literal.
              clientID:
              clientSecret:
              redirectURI: http://dex.auth.svc.cluster.local:5556/dex/callback
              orgs:
              # Fill in your GitHub organization name
              - name:
              # Required ONLY for GitHub Enterprise. Leave it empty when using github.com.
              # This is the Hostname of the GitHub Enterprise account listed on the
              # management console. Ensure this domain is routable on your network.
              hostName:
              # Flag which indicates that all user groups and teams should be loaded.
              loadAllGroups: false
              # flag which will switch from using the internal GitHub id to the users handle (@mention) as the user id.
              # It is possible for a user to change their own user name but it is very rare for them to do so
              useLoginAsID: false
        staticClients:
        - id: kubeflow-oidc-authservice
          redirectURIs: ["/login/oidc"]
          name: 'Dex Login Application'
          # Update the secret below to match with the oidc authservice.
          secret: pUBnBOY80SnXgjibTYM9ZWNzY2xreNGQok
    ```
    Save the `dex-cm.yaml` file.
    - Update this change to the Kubernetes cluster:
    ```
    kubectl apply -f dex-cm.yaml -n auth

    # Remove this file with sensitive information.
    rm dex-cm.yaml
    ```

1. Apply configuration changes:
    ```
    kubectl rollout restart deploy/dex -n auth
    ```

    Then visit the Kubeflow endpoint `<node_public_ip>:31380` to login into Kubeflow

1. Visit [KFP Tekton User Guide](/samples/kfp-user-guide) and start learning how to use Kubeflow pipeline.

1. Visit [KFP Tekton Admin Guide](/kfp-admin-guide.md) for how to configure kfp-tekton with different settings.

## Verify installation

1. Check the resources deployed correctly in namespace `kubeflow`

     ```
     kubectl get all -n kubeflow
     ```

1. Open Kubeflow Dashboard. The default installation does not create an external endpoint but you can use port-forwarding to visit your cluster. Run the following command and visit http://localhost:8080.

     ```
     kubectl port-forward svc/istio-ingressgateway -n istio-system 8080:80
     ```

In case you want to expose the Kubeflow Dashboard over an external IP, you can change the type of the ingress gateway. To do that, you can edit the service:

     kubectl edit -n istio-system svc/istio-ingressgateway

From that file, replace `type: NodePort` with `type: LoadBalancer` and save.

While the change is being applied, you can watch the service until below command prints a value under the `EXTERNAL-IP` column:

     kubectl get -w -n istio-system svc/istio-ingressgateway

The external IP should be accessible by visiting http://<EXTERNAL-IP>. Note that the above installation instructions do not create any protection for the external endpoint so it will be accessible to anyone without any authentication.

## Understanding the Kubeflow deployment process

The deployment process is controlled by the following commands:

* **build** - (Optional) Creates configuration files defining the various
  resources in your deployment. You only need to run `kfctl build` if you want
  to edit the resources before running `kfctl apply`.
* **apply** - Creates or updates the resources.
* **delete** - Deletes the resources.

### App layout

Your Kubeflow application directory **${KF_DIR}** contains the following files and
directories:

* **${CONFIG_FILE}** is a YAML file that defines configurations related to your
  Kubeflow deployment.

  * This file is a copy of the GitHub-based configuration YAML file that
    you used when deploying Kubeflow. For example, {{% config-uri-ibm %}}.
  * When you run `kfctl apply` or `kfctl build`, kfctl creates
    a local version of the configuration file, `${CONFIG_FILE}`,
    which you can further customize if necessary.

* **kustomize** is a directory that contains the kustomize packages for Kubeflow applications.
    * The directory is created when you run `kfctl build` or `kfctl apply`.
    * You can customize the Kubernetes resources (modify the manifests and run `kfctl apply` again).

You can find general information about Kubeflow configuration in the guide to [configuring Kubeflow with kfctl and kustomize](https://www.kubeflow.org/docs/other-guides/kustomize/).

## Troubleshooting
 - (For IBM Cloud IKS users) If you accidentally deployed Kubeflow with IBM Cloud File Storage, run the below commands to remove the existing pvc. The below commands are for removing resources in multi-user, so you can ignore any missing pvc or rollout error if you are doing this for single user.
    ```shell
    kubectl delete pvc -n kubeflow katib-mysql metadata-mysql minio-pv-claim minio-pvc mysql-pv-claim
    kubectl delete pvc -n istio-system authservice-pvc
    kubectl rollout restart -n kubeflow deploy/mysql deploy/minio deploy/katib-mysql deploy/metadata-db
    kubectl rollout restart -n istio-system statefulset/authservice
    ```

    Then, redo the [**single user**](#single-user) or [**multi-tenant**](#multi-user-auth-enabled) section to redeploy Kubeflow with the [block storageclass](https://www.kubeflow.org/docs/ibm/deploy/install-kubeflow/#ibm-cloud-block-storage-setup).

- If you redeploy Kubeflow and some components are not showing up, it was due to the [dynamic created webhook issue](https://github.com/kubeflow/manifests/issues/1379). This issue will be [fixed](https://github.com/kubeflow/pipelines/pull/4429) in the next release of KFP.
    ```shell
    kubectl delete MutatingWebhookConfiguration cache-webhook-kubeflow katib-mutating-webhook-config
    ```
