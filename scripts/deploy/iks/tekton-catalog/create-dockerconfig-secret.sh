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
 
REGISTRY_URL="${REGISTRY_URL:="https://index.docker.io/v1/"}"
DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:="kfptektonbot"}"
DOCKER_CONFIG_DIR="${DOCKER_CONFIG_DIR:="~/.docker"}"
SECRET_NAME="${SECRET_NAME:="registry-dockerconfig-secret"}"

# Create secret if not found
# NOTE - Could hold old credentials if created from a previous run
if ! kubectl get secret $SECRET_NAME
then
    echo "Creating docker-registry secret: $SECRET_NAME"
    kubectl create secret \
        docker-registry \
        $SECRET_NAME \
        --docker-server="${REGISTRY_URL}" \
        --docker-username="${DOCKERHUB_USERNAME}" \
        --docker-password="${DOCKERHUB_TOKEN}"
fi

# Save secret to local config file (for docker daemon to read credentials)
kubectl get secret $SECRET_NAME -o yaml \
    | yq -e eval '.data.".dockerconfigjson"' - \
    | base64 -d \
    > "${DOCKER_CONFIG_DIR}"/config.json

# Check that secret is not empty
if [ ! -s "${DOCKER_CONFIG_DIR}"/config.json ]
then
    echo "Error: ${DOCKER_CONFIG_DIR}/config.json is empty"
    exit 1
fi

echo "Secret written to ${DOCKER_CONFIG_DIR}/config.json"
