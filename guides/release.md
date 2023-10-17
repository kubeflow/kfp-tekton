# How to prepare for the KFP Tekton Release

1. Install [Kustomize V5](https://github.com/kubernetes-sigs/kustomize/releases/tag/kustomize%2Fv5.0.0).

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

# How to prepare for the KFP Tekton SDK Release

1. Set up Python virtual environment
    ```python
    python3 -m venv .venv
    source .venv/bin/activate
    ```

2. Go into the SDK package to build and publish the package to PYPI.
    ```python
    cd sdk/python
    pip install -e .
    export KFP_TEKTON_VERSION=${KFP_TEKTON_VERSION}
    twine check dist/kfp-tekton-${KFP_TEKTON_VERSION}.tar.gz
    twine upload --repository pypi dist/kfp-tekton-${KFP_TEKTON_VERSION}.tar.gz
    ```
