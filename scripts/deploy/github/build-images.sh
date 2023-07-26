#!/bin/bash
#
# Copyright 2023 kubeflow.org
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

docker system prune -a -f

docker build -q -t "${REGISTRY}/apiserver:latest" -f backend/Dockerfile . && docker push "${REGISTRY}/apiserver:latest"
docker build -q -t "${REGISTRY}/persistenceagent:latest" -f backend/Dockerfile.persistenceagent . && docker push "${REGISTRY}/persistenceagent:latest"
docker build -q -t "${REGISTRY}/scheduledworkflow:latest" -f backend/Dockerfile.scheduledworkflow . && docker push "${REGISTRY}/scheduledworkflow:latest"
docker build -q -t "${REGISTRY}/tekton-driver:latest" -f backend/Dockerfile.tektondriver . && docker push "${REGISTRY}/tekton-driver:latest" &
docker build -q -t "${REGISTRY}/tekton-exithandler-controller:latest" -f backend/Dockerfile.tekton-exithandler.controller . && docker push "${REGISTRY}/tekton-exithandler-controller:latest" &
docker build -q -t "${REGISTRY}/tekton-exithandler-webhook:latest" -f backend/Dockerfile.tekton-exithandler.webhook . && docker push "${REGISTRY}/tekton-exithandler-webhook:latest"
docker build -q -t "${REGISTRY}/tekton-kfptask-controller:latest" -f backend/Dockerfile.tekton-kfptask.controller . && docker push "${REGISTRY}/tekton-kfptask-controller:latest" &
docker build -q -t "${REGISTRY}/tekton-kfptask-webhook:latest" -f backend/Dockerfile.tekton-kfptask.webhook . && docker push "${REGISTRY}/tekton-kfptask-webhook:latest" &

wait

# clean up intermittent build caches to free up disk space
docker system prune -a -f
