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

echo "Attempting cleanup of kubernetes resources"

# At least make sure the manifests exist
[ -d tekton-catalog/pipeline-loops/config ]
[ -f tekton-catalog/pipeline-loops/examples/loop-example-basic.yaml ]

[ -f sdk/python/tests/compiler/testdata/any_sequencer.yaml ]

[ -f sdk/python/tests/compiler/testdata/resourceop_basic.yaml ]

# Pipeline Loops
kubectl delete -f tekton-catalog/pipeline-loops/config || true
kubectl delete -f tekton-catalog/pipeline-loops/examples/loop-example-basic.yaml || true

# Any Sequencer
kubectl delete -f sdk/python/tests/compiler/testdata/any_sequencer.yaml || true

# Kubectl Wrapper
kubectl delete -f sdk/python/tests/compiler/testdata/resourceop_basic.yaml || true

# Delete dockerhub credentials secret 
kubectl delete secret registry-dockerconfig-secret || true

# Jobs run in default namespace
kubectl delete job --all -n default || true
