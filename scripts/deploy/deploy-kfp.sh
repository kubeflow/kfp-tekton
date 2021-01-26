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

source scripts/deploy/helper_functions.sh

kubectl create ns $KUBEFLOW_NS

wait_for_namespace $KUBEFLOW_NS $MAX_RETRIES $SLEEP_TIME || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. \"${KUBEFLOW_NS}\" not found."
  exit $EXIT_CODE
fi

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

