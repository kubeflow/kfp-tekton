#!/bin/bash
# 
# Copyright 2021 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euxo pipefail

CONTROLLER_IMAGE_URL="${CONTROLLER_IMAGE_URL:=docker.io/aipipeline/pipelineloop-controller}"
WEBHOOK_IMAGE_URL="${WEBHOOK_IMAGE_URL:=docker.io/aipipeline/pipelineloop-webhook}"
IMAGE_TAG="${IMAGE_TAG:=nightly}"

CONTROLLER_MANIFEST="${CONTROLLER_MANIFEST:="config/500-controller.yaml"}"
CONTROLLER_IMAGE_PATH="${CONTROLLER_IMAGE_PATH:=".spec.template.spec.containers[0].image"}"
WEBHOOK_MANIFEST="${WEBHOOK_MANIFEST:="config/500-webhook.yaml"}"
WEBHOOK_IMAGE_PATH="${WEBHOOK_IMAGE_PATH:=".spec.template.spec.containers[0].image"}"

BUILD_DIR="tekton-catalog/pipeline-loops"

pushd $BUILD_DIR > /dev/null

# Ensure image fields exists
yq eval --exit-status ''"$CONTROLLER_IMAGE_PATH"'' $CONTROLLER_MANIFEST
# Webhook manifest is an array => filter to deployment
yq eval --exit-status ''"select(.kind == \"Deployment\") | ""$WEBHOOK_IMAGE_PATH"'' $WEBHOOK_MANIFEST

# Replace images inplace
yq eval --inplace ''"$CONTROLLER_IMAGE_PATH"=\"$CONTROLLER_IMAGE_URL:$IMAGE_TAG\"'' $CONTROLLER_MANIFEST
yq eval --inplace ''"select(.kind == \"Deployment\") |= ""$WEBHOOK_IMAGE_PATH"=\"$WEBHOOK_IMAGE_URL:$IMAGE_TAG\"'' $WEBHOOK_MANIFEST

echo "========== CONTROLLER MANIFEST =========="
cat $CONTROLLER_MANIFEST
echo "========================================="

echo "========== WEBHOOK MANIFEST =========="
cat $WEBHOOK_MANIFEST
echo "======================================"

# Add imagePullSecrets to service accounts
yq eval --inplace '.imagePullSecrets[0].name = "all-icr-io"' config/200-serviceaccount.yaml

echo "========== SERVICE ACCOUNT MANIFEST =========="
cat config/200-serviceaccount.yaml
echo "=============================================="

popd > /dev/null
