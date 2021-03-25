# IBM Cloud Toolchain for [kfp-tekton](https://github.com/kubeflow/kfp-tekton)

We use the "Develop a Kubernetes app" [toolchain](https://www.ibm.com/cloud/architecture/tutorials/use-develop-kubernetes-app-toolchain?task=2) to enable CI/CD testing for kfp-tekton.

![workflow](workflow.png)

The toolchain is bound to an IBM Cloud Kubernetes Service (IKS) cluster and runs the following in a *delivery pipeline*:

1. Pull the latest commit from github
2. Run unit tests
3. Build docker images and push to IBM Cloud Registry
3. Deploy tekton and kfp-tekton to IKS
4. Run e2e tests (submit a pipeline, check endpoints, etc)
5. Remove tekton and kfp-tekton

## Custom Docker Image
In order to build, test, deploy, etc... within the pipeline, we use a `Custom Docker Image` to package all the requisite build and runtime dependencies. The pipeline takes as input the [Dockerfile](./Dockerfile), which contains:
- Node.js
- Go
- kubectl
- kustomize
- helm2
- heml3
- jq
- ibmcloud

Run the following command to build the image:
```
docker build -t pipeline-base-image -f Dockerfile .
```

Or you can build and push the image to ibm cloud container registry:
```
ibmcloud cr build -f Dockerfile --tag <registry_url>/<namespace>/pipeline-base-image:<image tag> .
```
Replace the `registry url`, `namespace`, `image tag`, and even the `image name` as needed.

**Note:**
You can also use docker arguments to specify the version of `Node.js`, `Go`, `kubectl`, `heml2`, `heml3`, etc. Check the `ARG` in Dockerfile to see the details.

## Scripts

When running jobs in a pipeline, you can `source` an external shell script. For example:
```
source <(curl -sSL "https://raw.githubusercontent.com/yhwang/kfp-tekton-toolchain/main/scripts/run-test.sh")
```

The following scripts are used within the pipeline:
- `run-test.sh`
  - Runs `kfp-tekton`'s unit tests.
- `build-image.sh`
  - Cleans up kfp-tekton docker images in the container registry and builds the kfp-tekton docker images: `api-server`, `persistenceagent`, `metadata-writer`, and `scheduledworkflow`.
  - The environment variables `DOCKER_FILE`, `DOCKER_ROOT`, `IMAGE_NAME` must be specified properly. For example, use `DOCKER_FILE=backend/Dockerfile` and `IMAGE_NAME=api-server` to build the `api-server` image. It also needs some variables from `run-test.sh` script. The
  script only builds one image according to the `DOCKER_FILE` and `IMAGE_NAME` specified. In order to build all the images, you need to create multiple jobs and assign different
  values for those environment variables.
- `deploy-tekton.sh`
  - Deploys `tekton` to the cluster.
- `deploy-kfp-tekton.sh`
  - Creates the `kubeflow` namespace and deploys `kfp-tekton` to the cluster.
- `e2e.sh`
  - Runs an "end-to-end" test. The `flip coin` pipeline is used. The pipeline is uploaded to kubeflow, executed, and checked for a passing result.
- `undeploy-kubeflow.sh`
  - Deletes the kubeflow deployment and cleans up any related resources.
- `undeploy-tekton.sh`
  - Deletes the tekton deployment and cleans up any related resources.
- `deploy-ibm-vpc.sh` and `./deploy-kfp-ibm-vpc.sh` 
  - Please see the section [A script to deploy kubernetes cluster to IBM Cloud IKS.](#a-script-to-deploy-kubernetes-cluster-to-ibm-cloud-iks), 
    for usage guide.

These scripts store variables into ${ARCHIVE_DIR}/build.properties which could be used
by the subsequent jobs in the next stage. You need to specify `build.properties` as a
property file in the `Environment properties` tab.

## Status

WIP-

The toolchain is bound to XXX cluster.

The toolchain executes every XXX.

The toolchain outputs job updates to [this](https://ibm-cloudplatform.slack.com/archives/G01LD87L81Z) slack channel.

## A script to deploy kubernetes cluster to IBM Cloud IKS.

### Goals:

1. Provide CLI options to start a IBM Cloud vpc-gen2 IKS cluster with kubeflow deployed, in just one step.
2. Reasonable defaults for every option, allowing the user to spend least amount of time understanding how it works.
3. Takes care of complete cleanup of resources or just delete the cluster.
4. A cluster may be deployed in an existing VPC.
5. Store the cluster state, so that next run remembers the resources and perform delete/ or start new cluster operation
   in the previously created VPC.
6. If a run crashes, allow for resuming where it left off, when re-run with same CLI options as previous run.

### Non Goals:
1. Managing user login. i.e. before running the script, user should be logged in or script will exit with suitable error.

### User guide
Know more about the CLI options (and the defaults) by executing: 
`./deploy-ibm-vpc.sh --help`

*Please note, the script stores all the CLI options passed as configuration for future run. The next run
will load those values as default values for the specified CLI options. The location of the config can be specified by,
--config-file="/path/config-file"*

1. Deploy cluster with just defaults.

  ```shell
    ./deploy-ibm-vpc.sh
    ... (wait for finish.)
    kubectl get nodes
    kubectl -n kubeflow get pods
    ... (should list all the pods as running.)
  ```
   __The above run does not provide any CLI options (e.g. --cluster-name/--vpc-name). The script above can either
   generate a cluster-name and vpc-name on it's own or load these values from config-file.__

2. Deploy cluster with cluster-name and vpc-name provided.

  ```shell
  ./deploy-ibm-vpc.sh --cluster-name="my-cluster" --vpc-name="my-vpc"
   ... (wait for finish.)
  kubectl get nodes
  kubectl -n kubeflow get pods
  ```
3. Delete a cluster, that was previously started.
  
   ```shell
   ./deploy-ibm-vpc.sh --delete-cluster="cluster"
   ```
  
   __The above step, deletes a cluster by looking up previously stored state in config file
   (i.e. the option --config-file=`/path/value`)__

4. Start again cluster.
   
   ```shell
   ./deploy-ibm-vpc.sh
    ... (wait for finish.) A cluster with same specification will be started as specified in previous run.
   ```
5. Delete a cluster by specifying it's name and name of the VPC where it is running.
  
  ```shell
  ./deploy-ibm-vpc.sh --cluster-name="my-cluster" --vpc-name="my-vpc" --delete-cluster="cluster"
  ```

6. Completely delete cluster and associated resources.

  Please note that, a subnet cannot be deleted unless it is released by VNIC that is attached to the instance. This can
  only, happen when that instance is deleted. At the moment there is no mechanism provided by cloud API to detach a VNIC
  from a subnet.
  
  Similarly, a VPC can only be delete when all it's resources are released. When a cluster is deleted it takes more
  than 40 mins, to completely release it's resources. And, it is not a great user experience if the scripts hangs for
  40 mins to await a complete delete of the cluster. The other alternative is to prompt the user to run the same delete
  command again, once the cluster is deleted. Second attempt of delete will make sure any lingering resources are also
  released.
  
    ```shell 
    ./deploy-ibm-vpc.sh --delete-cluster="full"
     (It takes a while for the cluster to get deleted and all the resources released. You may run
     the script again with same option (i.e.  --delete-cluster="full") to reattempt delete.
    ```
  
  __A "full" delete also deletes the config-file, the stored state (i.e. VPC id/subnets/clusters etc.) of a vpc is
  irrelevant once the VPC is deleted.__
