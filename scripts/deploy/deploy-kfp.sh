#!/bin/bash
#
# Copyright 2020 kubeflow.org
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

set -ex

MAX_RETRIES="${MAX_RETRIES:-5}"
SLEEP_TIME="${SLEEP_TIME:-10}"
EXIT_CODE=0

MANIFEST_DIR="${MANIFEST_DIR:-"manifests/kustomize/env/platform-agnostic"}"
KUBEFLOW_NS="${KUBEFLOW_NS:-kubeflow}"

API_SERVER_IMAGE="${API_SERVER_IMAGE:-"docker.io/aipipeline/api-server"}"
NEW_API_SERVER_IMAGE="${NEW_API_SERVER_IMAGE:-"us.icr.io/kfp-tekton/api-server"}"

METADATA_WRITER_IMAGE="${METADATA_WRITER_IMAGE:-"docker.io/aipipeline/metadata-writer"}"
NEW_METADATA_WRITER_IMAGE="${NEW_METADATA_WRITER_IMAGE:-"us.icr.io/kfp-tekton/metadata-writer"}"

PERSISTENCEAGENT_IMAGE="${PERSISTENCEAGENT_IMAGE:-"docker.io/aipipeline/persistenceagent"}"
NEW_PERSISTENCEAGENT_IMAGE="${NEW_PERSISTENCEAGENT_IMAGE:-"us.icr.io/kfp-tekton/persistenceagent"}"

SCHEDULEDWORKFLOW_IMAGE="${SCHEDULEDWORKFLOW_IMAGE:-"docker.io/aipipeline/scheduledworkflow"}"
NEW_SCHEDULEDWORKFLOW_IMAGE="${NEW_SCHEDULEDWORKFLOW_IMAGE:-"us.icr.io/kfp-tekton/scheduledworkflow"}"

source scripts/deploy/helper_functions.sh

kubectl create ns $KUBEFLOW_NS

wait_for_namespace $KUBEFLOW_NS $MAX_RETRIES $SLEEP_TIME || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. \"${KUBEFLOW_NS}\" not found."
  exit $EXIT_CODE
fi

# Edit image names in kustomize files 
# No need to build as long as deployed with "kubectl apply -k"
pushd manifests/kustomize/env/platform-agnostic > /dev/null

kustomize edit set image $API_SERVER_IMAGE=$NEW_API_SERVER_IMAGE
kustomize edit set image $METADATA_WRITER_IMAGE=$NEW_METADATA_WRITER_IMAGE
kustomize edit set image $PERSISTENCEAGENT_IMAGE=$NEW_PERSISTENCEAGENT_IMAGE
kustomize edit set image $SCHEDULEDWORKFLOW_IMAGE=$NEW_SCHEDULEDWORKFLOW_IMAGE

popd > /dev/null

# Deploy manifest

deploy_with_retries $MANIFEST_DIR $MAX_RETRIES $SLEEP_TIME || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. Failure applying $MANIFEST_DIR."
  exit 1
fi

# Check if all pods are running - allow 60 retries (10 minutes)

wait_for_pods $KUBEFLOW_NS 60 $SLEEP_TIME || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. Not all pods running."
  exit 1
fi

echo "Finished kfp-tekton deployment." 

