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

BUILD_DIR="sdk/python/tests/compiler/testdata"

OLD_IMAGE="dspipelines/any-sequencer:latest"
NEW_IMAGE_URL="${NEW_IMAGE_URL:=us.icr.io/kfp-tekton/any-sequencer}"
NEW_IMAGE_TAG="${NEW_IMAGE_TAG:=nightly}"

MANIFEST="any_sequencer.yaml"

pushd $BUILD_DIR > /dev/null

if ! grep -q "image: $OLD_IMAGE" $MANIFEST; then
    echo "Unable to find $OLD_IMAGE in any_sequencer.yaml"
    exit 1
fi

# Use | as delimiter to avoid issues with slashes.
sed 's|'"$OLD_IMAGE"'|'"$NEW_IMAGE_URL:$NEW_IMAGE_TAG"'|g' $MANIFEST > any_sequencer_temp.yaml

mv any_sequencer_temp.yaml $MANIFEST

echo "========== MANIFEST =========="
cat $MANIFEST
echo "=============================="

# Note - Permission to watch status required: "kubectl create clusterrolebinding pipeline-runner-extend --clusterrole=cluster-admin --serviceaccount=default:default"
kubectl apply -f $MANIFEST

wait_for_pipeline_run "any-sequencer" 20 30

kubectl delete -f $MANIFEST

popd > /dev/null
