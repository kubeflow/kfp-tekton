# Tekton KFP Changelog

## Tekton 0.13 release
- Updated go.mod with the new go-client dependencies
- Replaced k8s.io/kubernetes to k8s.io/api due to new 1.17 go-client requirements
- Update persistent agent Docker image to be based on golang 1.13 because go-client 1.17.6 is no longer supported on golang 1.11

## Initial commit

- \[Backend\] init Tekton backend commit [\#4](https://github.com/kubeflow/kfp-tekton-backend/pull/4)
    - Modified api docker build to use go modules
    - Moved all the Argo specific API to Tekton specific API (Still need to implement it with polymorphism or abstracted function as the ideal solution)
        - workflow type changed to pipelineRun type
        - Argo v1alpha1 api changed to Tekton v1beta1 api
        - param value type changed from Argo string to Tekton ArrayOrString Type
    - go.mod is updated to match with the Tekton dependencies.
    - go.sum is updated for dependency version control.
    - Commented out Argo specific tests to skip type check errors.
