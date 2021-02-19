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

KUBECTL_WRAPPER_IMAGE_URL="${KUBECTL_WRAPPER_IMAGE_URL:=docker.io/aipipeline/kubeclient}"
IMAGE_TAG="${IMAGE_TAG:=nightly}"

BUILD_DIR="sdk/python/tests/compiler/testdata"
MANIFEST="${MANIFEST:="resourceop_basic.yaml"}"

pushd $BUILD_DIR > /dev/null

# Ensure image fields exists
yq eval --exit-status '.spec.pipelineSpec.tasks[0].taskSpec.params[] | select(.name == "image") | .default' $MANIFEST

# Replace images inplace
yq eval --inplace ''".spec.pipelineSpec.tasks[0].taskSpec.params[] |= select(.name == \"image\") |= .default = \"$KUBECTL_WRAPPER_IMAGE_URL:$IMAGE_TAG\""'' $MANIFEST

popd > /dev/null
