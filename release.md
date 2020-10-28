# How to prepare for the KFP Tekton Release

1. Install [Kustomize V3](https://kubernetes-sigs.github.io/kustomize/installation/).

2. Update the [kfp tekton manifest template](/manifests/kustomize/env/kfp-template) for any kustomization update on this release.

3. Generate the KFP-Tekton deployment yaml for the release.
    ```shell
    export KFP_TEKTON_RELEASE=<release_tag>
    make build-release-template
    ```

4. Test the generated yaml to verify if it works properly
    ```shell
    kubectl apply -f install/${KFP_TEKTON_RELEASE}/kfp-tekton.yaml
    ```
