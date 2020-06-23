# Installing Kubeflow Pipelines with Tekton

# Getting Started
## Prequisites
1. [Install Tekton](https://github.com/tektoncd/pipeline/blob/master/docs/install.md#installing-tekton-pipelines-on-kubernetes) v0.13.0 or above
2. [Install Kubeflow](https://www.kubeflow.org/docs/started/getting-started/) if you want to leverage the Kubeflow stack
3. Clone this repository
    ```
    git clone github.com/kubeflow/kfp-tekton
    cd kfp-tekton
    ```

## Install Tekton KFP with pre-built images
1. Remove the old version of KFP from a previous Kubeflow deployment if it exists on your cluster. Also clean up the old webhooks if you were using KFP 0.4 or above.
    ```shell
    kubectl delete -k manifests/kustomize/env/platform-agnostic
    kubectl delete MutatingWebhookConfiguration cache-webhook-kubeflow
    ```

2. Once the old KFP deployment is removed, run the below command to deploy this modified Kubeflow Pipeline version with Tekton backend.
    ```shell
    kubectl apply -k manifests/kustomize/env/platform-agnostic
    ```

    Check the new KFP deployment, it should take about 5 to 10 minutes.
    ```shell
    kubectl get pods -n kubeflow
    ```

    Now go ahead and access the pipeline in the Kubeflow dashboard. It should be accessible from the istio-ingressgateway which is the
    `<public_ip>:31380`

## Development: Building from source code
### Prerequites
1. [NodeJS 12 or above](https://nodejs.org/en/download/)
2. [Golang 1.13 or above](https://golang.org/dl/)

### Frontend
The development instructions are under the [frontend](/frontend) directory. Below are the commands for building the frontend docker image.
```shell
cd frontend
npm run docker
```

### Backend
The KFP backend with Tekton needs to modify the Kubeflow Pipelines api-server and persistent agent. 
1. To build these two images, clone this repository under the [GOPATH](https://golang.org/doc/gopath_code.html#GOPATH) and rename it to `pipelines`. 
    ```shell
    cd $GOPATH/src/go/github.com/kubeflow
    git clone github.com/kubeflow/kfp-tekton
    mv kfp-tekton pipelines
    cd pipelines
    ```

2. For local binary builds, use the `go build` commands
   ```shell
   go build -o apiserver ./backend/src/apiserver
   go build -o agent ./backend/src/agent/persistence
   ```

3. For Docker builds, use the below `docker build` commands
   ```shell
   docker build -t api-server -f backend/Dockerfile .
   docker build -t persistenceagent -f backend/Dockerfile.persistenceagent .
   ```

4. Push the images to registry and modify the Kustomization to use your own built images.
    
   Modify the `newName` under the `images` section in `manifests/kustomize/base/kustomization.yaml`.

   Now you can follow the [Install Tekton KFP with pre-built images](#install-tekton-kfp-with-pre-built-images) instructions to install your own KFP backend.
