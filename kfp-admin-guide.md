# Kubeflow Pipeline Admin guide for Tekton backend.

This page introduces different ways to configure the kfp-tekton admin settings such as configuring artifacts, log archival, and auto strip EOF newlines for Tekton results. Below are the list of settings for kfp-tekton admin. The default settings for kfp-tekton are located at [here](/manifests/kustomize/env/platform-agnostic/kfp-pipeline-config.yaml).

## Table of Contents

- [Disable Artifact Tracking](#disable-artifact-tracking)
- [Enable Log Archival](#enable-log-archival)
- [Enable Auto Strip for End of File newlines](#enable-auto-strip-for-end-of-file-newlines)
- [Customize Artifact Image to do your own post processing](#customize-artifact-image-to-do-your-own-post-processing)


## Disable Artifact Tracking

By default, kfp-tekton enabled artifacts for archiving the pipeline outputs and use it for metadata tracking. To disable this feature, run the following commands to update the configmap and rollout a new server.

```shell
kubectl patch cm kfp-tekton-config -n kubeflow -p '{"data":{"track_artifacts":"false"}}'
kubectl rollout restart deploy/ml-pipeline -n kubeflow
```

## Enable Log Archival

Log Archival will capture the log from each task and archived to the artifact storage as an output artifact. By default this feature is disabled. To enable this feafure, run the following commands:

```shell
kubectl patch cm kfp-tekton-config -n kubeflow -p '{"data":{"archive_logs":"true"}}'
kubectl rollout restart deploy/ml-pipeline -n kubeflow
kubectl rollout restart deploy/metadata-writer -n kubeflow
```

## Enable Auto Strip for End of File newlines

Tekton by design are passing parameter outputs as it including unintentional End of File (EOF) newlines. Tekton are expecting users to know this behavior when designing their components. Therefore, the kfp-tekton team designed an experimental feature to auto strip the EOF newlines for a better user experience. This feature is disabled by default and only works for files that are not depended on EOF newlines. To enable this feature, run the following commands:
```shell
kubectl patch cm kfp-tekton-config -n kubeflow -p '{"data":{"strip_eof":"true"}}'
kubectl rollout restart deploy/ml-pipeline -n kubeflow
```

## Customize Artifact Image to do your own post processing

Since Tekton still has many gaps with handling artifacts, KFP-Tekton allows users to provide their own artifact image for post-processing and pushing their artifacts. By default, KFP-Tekton uses the minio/mc image for pushing artifacts due to its lightweight design (10MB size). Since the minio/mc image is running with the bare minimum kernel, it requires the KFP-Tekton to process the artifact annotations into basic sh commands and inject it as part of the script. To add you own post processing, please read the definition below and customize any flag as needed.

- `artifact_image`: Image for processing and pushing the artifacts
- `artifact_script`: Entrypoint script for running the artifact image.
- `inject_default_script`: A set of default script that convert the artifact annotations into `sh` script. We recommend to disable it if using a custom artifact image. 

Then update the default [kfp-tekton configmap](/manifests/kustomize/env/platform-agnostic/kfp-pipeline-config.yaml) and patch it with the commands below:
```
kubectl apply -f manifests/kustomize/env/platform-agnostic/kfp-pipeline-config.yaml
kubectl rollout restart deploy/ml-pipeline -n kubeflow
```

