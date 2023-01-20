#!/bin/bash
#
# Copyright 2021 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Remove the x if you need no print out of each command
set -e

# Need the following env
# - PIPELINE_KUBERNETES_CLUSTER_NAME:   kube cluster name
# - KUBEFLOW_NS:                        namespace for kfp-tekton, defulat: kubeflow

MAX_RETRIES="${MAX_RETRIES:-5}"
SLEEP_TIME="${SLEEP_TIME:-30}"
REGISTRY="${REGISTRY:-kind-registry:5000}"
EXIT_CODE=0

KUSTOMIZE_DIR="${KUSTOMIZE_DIR:-"manifests/kustomize/env/platform-agnostic-kind"}"
MANIFEST="${MANIFEST:-"kfp-tekton.yaml"}"
KUBEFLOW_NS="${KUBEFLOW_NS:-kubeflow}"

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then C_DIR="$PWD"; fi
source "${C_DIR}/../iks/helper-functions.sh"

# Download kustomize
wget --quiet --output-document=./kustomize https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.2.1/kustomize_kustomize.v3.2.1_linux_amd64 \
  && chmod +x ./kustomize

kubectl create ns "$KUBEFLOW_NS"

wait_for_namespace "$KUBEFLOW_NS" "$MAX_RETRIES" "$SLEEP_TIME" || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. \"${KUBEFLOW_NS}\" not found."
  exit $EXIT_CODE
fi

pushd "$KUSTOMIZE_DIR" > /dev/null
kustomize edit set image "docker.io/aipipeline/api-server=${REGISTRY}/kfp-tekton/apiserver:latest"
kustomize edit set image "docker.io/aipipeline/persistenceagent=${REGISTRY}/kfp-tekton/persistenceagent:latest"
kustomize edit set image "docker.io/aipipeline/metadata-writer=${REGISTRY}/kfp-tekton/metadata-writer:latest"
kustomize edit set image "docker.io/aipipeline/scheduledworkflow=${REGISTRY}/kfp-tekton/scheduledworkflow:latest"
kustomize edit set image "docker.io/aipipeline/cache-server=${REGISTRY}/kfp-tekton/cache-server:latest"
kustomize edit set image "docker.io/aipipeline/frontend=${REGISTRY}/kfp-tekton/frontend:latest"

popd > /dev/null

# Build manifest
./kustomize build "$KUSTOMIZE_DIR" -o "${MANIFEST}"

# Deploy manifest
deploy_with_retries "-f" "${MANIFEST}" "$MAX_RETRIES" "$SLEEP_TIME" || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. Failure applying $KUSTOMIZE_DIR."
  exit 1
fi


# Check if all pods are running - allow 60 retries (10 minutes)

wait_for_pods "$KUBEFLOW_NS" 20 "$SLEEP_TIME" || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. Not all pods running."
  exit 1
fi

echo "Finished kfp-tekton deployment."
