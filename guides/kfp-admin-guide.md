# Kubeflow Pipeline Admin guide for Tekton backend.

This page introduces different ways to configure the kfp-tekton admin settings such as configuring artifacts, log archival, and auto strip EOF newlines for Tekton results. Below are the list of settings for kfp-tekton admin. The default settings for kfp-tekton are located at [here][kfp-tekton-configmap].

## Table of Contents

- [Disable Artifact Tracking](#disable-artifact-tracking)
- [Disable Log Archival](#disable-log-archival)
- [Disable Auto Strip for End of File newlines](#disable-auto-strip-for-end-of-file-newlines)
- [Customize Artifact Image to do your own post processing](#customize-artifact-image-to-do-your-own-post-processing)
- [Customize S3 Endpoint for KFP Tekton artifacts](#customize-s3-endpoint-for-kfp-tekton-artifacts)
- [Disable Auto Apply for Tekton Custom Resources](#disable-auto-apply-for-tekton-custom-resources)
- [Disable Caching](#disable-caching)
- [Change the Tekton Terminate API Method](#change-the-tekton-terminate-api-method)


## Disable Artifact Tracking

By default, kfp-tekton enabled artifacts for archiving the pipeline outputs and use it for metadata tracking. To disable this feature, run the following commands to update the configmap and rollout a new server.

```shell
kubectl patch cm kfp-tekton-config -n kubeflow -p '{"data":{"track_artifacts":"false"}}'
kubectl rollout restart deploy/ml-pipeline -n kubeflow
```

## Disable Log Archival

Log Archival will capture the log from each task and archived to the artifact storage as an output artifact. By default this feature is enabled. To disable this feafure, run the following commands:

```shell
kubectl patch cm kfp-tekton-config -n kubeflow -p '{"data":{"archive_logs":"false"}}'
kubectl rollout restart deploy/ml-pipeline -n kubeflow
kubectl rollout restart deploy/metadata-writer -n kubeflow
```

Archived logs are enabled in the KFP-Tekton UI by default. To disable this feature, run the following command:

```shell
kubectl set env -n kubeflow deploy/ml-pipeline-ui ARGO_ARCHIVE_LOGS=false
```

## Disable Auto Strip for End of File newlines

Tekton by design are passing parameter outputs as it including unintentional End of File (EOF) newlines. Tekton are expecting users to know this behavior when designing their components. Therefore, the kfp-tekton team designed an experimental feature to auto strip the EOF newlines for a better user experience. This feature is enabled by default and only works for files that are not depended on EOF newlines. To disable this feature, run the following commands:
```shell
kubectl patch cm kfp-tekton-config -n kubeflow -p '{"data":{"strip_eof":"false"}}'
kubectl rollout restart deploy/ml-pipeline -n kubeflow
```

## Customize Artifact Image to do your own post processing

Since Tekton still has many gaps with handling artifacts, KFP-Tekton allows users to provide their own artifact image for post-processing and pushing their artifacts. By default, KFP-Tekton uses the minio/mc image for pushing artifacts due to its lightweight design (10MB size). Since the minio/mc image is running with the bare minimum kernel, it requires the KFP-Tekton to process the artifact annotations into basic sh commands and inject it as part of the script. To add you own post processing, please read the definition below and customize any flag as needed.

- `artifact_image`: Image for processing and pushing the artifacts
- `artifact_script`: Entrypoint script for running the artifact image.
- `inject_default_script`: A set of default script that convert the artifact annotations into `sh` script. We recommend to disable it if using a custom artifact image.

Then update the default [kfp-tekton configmap][kfp-tekton-configmap] and patch it with the commands below:
```shell
kubectl apply -f /manifests/kustomize/base/pipeline/kfp-pipeline-config.yaml -n kubeflow
kubectl rollout restart deploy/ml-pipeline -n kubeflow
```

## Customize S3 Endpoint for KFP Tekton artifacts

By default, Kubeflow Pipelines store pipeline artifacts to the S3 compatible Minio storage server in the same Kubernetes cluster. To change this storage to an external S3 endpoint, update the default [kfp-tekton configmap][kfp-tekton-configmap] and S3 credential secret then patch them with the commands below.

Update the S3 Secret:
- `accesskey`: S3 Access Key ID
- `secretkey`: S3 Secret Access Key
```shell
# Define the S3 access and secret key
accesskey=<s3_access_key>
secretkey=<s3_secret_access_key>

kubectl create secret generic -n kubeflow mlpipeline-minio-artifact --from-literal=accesskey=${accesskey}  --from-literal=secretkey=${secretkey} --dry-run -o yaml | kubectl apply -f -
```

Update the [kfp-tekton configmap][kfp-tekton-configmap]:
- `artifact_bucket`: Bucket for storing the artifacts
- `artifact_endpoint`: S3 Endpoint for storing artifacts
- `artifact_endpoint_scheme`: HTTP scheme of the above S3 Endpoint
```shell
kubectl apply -f manifests/kustomize/env/platform-agnostic/kfp-pipeline-config.yaml -n kubeflow
kubectl rollout restart deploy/ml-pipeline -n kubeflow
kubectl rollout restart deploy/minio -n kubeflow
```

## Disable Auto Apply for Tekton Custom Resources

Due to the lack of inline task template support in the current Tekton API, we allow the SDK to embed any custom resource to the pipelineRun annotations and apply it for every pipeline run. However, all the custom resources generated by the SDK are only unique for each pipeline version but not for each pipeline run. This is due to all Tekton resources are namespace-scope, and all pipeline runs share the same namespace.

Although this feature allows users to deploy all the necessary custom resources with one KFP API call, KFP doesn't clean up these custom resources because the same custom resource could have multiple owners. This feature mainly is for recursion loops where the sub-pipeline needs to be self-referencible.

If the admin doesn't want to let kfp-tekton handles the creation of Tekton custom resources, this feature can be disabled with:
```shell
kubectl patch cm kfp-tekton-config -n kubeflow -p '{"data":{"apply_tekton_custom_resource":"false"}}'
kubectl rollout restart deploy/ml-pipeline -n kubeflow
```

To disable it from the SDK as a user, add the following Tekton config during compile time.
```python
pipeline_conf = TektonPipelineConf()
pipeline_conf.set_resource_in_separate_yaml(True)
self._test_pipeline_workflow(test_pipeline, 'test.yaml', tekton_pipeline_conf=pipeline_conf)
```

## Disable Caching

KFP Caching will cache all the workloads that use the same task with the same inputs. It's enabled by default. To disable caching on the server side, run the commands below.

```shell
kubectl set env -n kubeflow deploy/ml-pipeline CACHE_ENABLED=false
```

## Change the Tekton Terminate API Method

Starting from Tekton 0.25.0, there are three different ways to terminate a pipeline in Tekton.
- "StoppedRunFinally" - To stop (i.e. let the tasks complete, then execute finally tasks) a PipelineRun
- "CancelledRunFinally" - To cancel (i.e. interrupt any executing non finally tasks, then execute finally tasks)
- "Cancelled" - Interrupt any executing tasks without running finally tasks

By default, kfp-tekton uses `Cancelled` as the terminate API for Tekton. However, if the admin wants to change the terminate API to other methods, the admin can update the default [kfp-tekton configmap][kfp-tekton-configmap] and apply the configamap with the below commands

- `terminate_status`: Tekton terminate status for the KFP terminate API. Default is `Cancelled`.

```shell
kubectl apply -f /manifests/kustomize/base/pipeline/kfp-pipeline-config.yaml -n kubeflow
kubectl rollout restart deploy/ml-pipeline -n kubeflow
```


[kfp-tekton-configmap]: /manifests/kustomize/base/pipeline/kfp-pipeline-config.yaml
