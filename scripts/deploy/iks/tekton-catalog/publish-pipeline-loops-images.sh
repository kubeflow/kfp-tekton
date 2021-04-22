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

DOCKER_REGISTRY="${DOCKER_REGISTRY:=docker.io}"
DOCKER_NAMESPACE="${DOCKER_NAMESPACE:=aipipeline}"
IMAGE_TAG="${IMAGE_TAG:=nightly}"

BUILD_DIR="tekton-catalog/pipeline-loops"

pushd $BUILD_DIR > /dev/null

docker build --build-arg bin_name=pipelineloop-controller . -t ${DOCKER_NAMESPACE}/pipelineloop-controller:"${IMAGE_TAG}"
docker build --build-arg bin_name=pipelineloop-webhook . -t ${DOCKER_NAMESPACE}/pipelineloop-webhook:"${IMAGE_TAG}"

docker push ${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/pipelineloop-controller:"${IMAGE_TAG}"
docker push ${DOCKER_REGISTRY}/${DOCKER_NAMESPACE}/pipelineloop-webhook:"${IMAGE_TAG}"

popd > /dev/null
