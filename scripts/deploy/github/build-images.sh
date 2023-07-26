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
# source: https://raw.githubusercontent.com/open-toolchain/commons/master/scripts/check_registry.sh

# Remove the x if you need no print out of each command
set -e

REGISTRY="${REGISTRY:-kind-registry:5000}"

docker build -t "${REGISTRY}/kfp-tekton/apiserver:latest" -f backend/Dockerfile . && docker push "${REGISTRY}/kfp-tekton/apiserver:latest" &
docker build -t "${REGISTRY}/kfp-tekton/persistenceagent:latest" -f backend/Dockerfile.persistenceagent . && docker push "${REGISTRY}/kfp-tekton/persistenceagent:latest" &
docker build -t "${REGISTRY}/kfp-tekton/metadata-writer:latest" -f backend/metadata_writer/Dockerfile . && docker push "${REGISTRY}/kfp-tekton/metadata-writer:latest" &
docker build -t "${REGISTRY}/kfp-tekton/scheduledworkflow:latest" -f backend/Dockerfile.scheduledworkflow . && docker push "${REGISTRY}/kfp-tekton/scheduledworkflow:latest" &
docker build -t "${REGISTRY}/kfp-tekton/cache-server:latest" -f backend/Dockerfile.cacheserver . && docker push "${REGISTRY}/kfp-tekton/cache-server:latest" &
docker build -t "${REGISTRY}/kfp-tekton/frontend:latest" -f frontend/Dockerfile . && docker push "${REGISTRY}/kfp-tekton/frontend:latest" &
wait

docker system prune -a -f