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

source scripts/deploy/iks/helper-functions.sh

BUILD_DIR="tekton-catalog/pipeline-loops"

E2E_MANIFEST="${E2E_MANIFEST:=examples/loop-example-basic.yaml}"

pushd $BUILD_DIR > /dev/null

# Note - Copy secret to tekton-pipelines namespace prior to this step: "kubectl get secret all-icr-io -n default -o yaml | sed "s/default/${NAMESPACE}/g” | kubectl apply -f -"
# ignore the error if the secret already exists
kubectl get secret all-icr-io -n default -o yaml | sed "s/default/tekton-pipelines/g" | kubectl apply -f - || true

kubectl apply -f config

wait_for_pod "tekton-pipelines" "tekton-pipelineloop-controller" 5 5
wait_for_pod "tekton-pipelines" "tekton-pipelineloop-webhook" 5 5

kubectl apply -f ${E2E_MANIFEST}

wait_for_pipeline_run "pr-loop-example" 20 30

kubectl delete -f ${E2E_MANIFEST}

# Clean up deployment
kubectl delete -f config

popd > /dev/null
