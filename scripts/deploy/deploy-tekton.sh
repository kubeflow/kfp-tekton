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

TEKTON_NS="${TEKTON_NS:-"tekton-pipelines"}"
# Previous versions use form: "previous/vX.Y.Z"
TEKTON_VERSION="${TEKTON_VERSION:-"latest"}"
TEKTON_MANIFEST="${TEKTON_MANIFEST:-https://storage.googleapis.com/tekton-releases/pipeline/${TEKTON_VERSION}/release.yaml}"

source scripts/deploy/helper_functions.sh

kubectl apply -f $TEKTON_MANIFEST

wait_for_namespace $TEKTON_NS $MAX_RETRIES $SLEEP_TIME || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. \"${TEKTON_NS}\" not found."
  exit $EXIT_CODE
fi

wait_for_pods $TEKTON_NS $MAX_RETRIES $SLEEP_TIME || EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]
then
  echo "Deploy unsuccessful. Not all pods running."
  exit 1
fi

echo "Finished tekton deployment."

