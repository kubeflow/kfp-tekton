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

TEKTON_MANIFEST="${TEKTON_MANIFEST:-https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml}"

kubectl delete -f $TEKTON_MANIFEST

echo "Finished tekton undeployment."

