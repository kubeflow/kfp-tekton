#!/usr/bin/env bash

# Copyright 2019 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# run this on the project root directory

set -o errexit
set -o nounset
set -o pipefail

# need these two go pkgs for code-gen:
#  - "knative.dev/hack"
#	 - "knative.dev/pkg/hack"

CODE_GEN_DIR=code-gen

go mod vendor
# If we run with -mod=vendor here, then generate-groups.sh looks for vendor files in the wrong place.
export GOFLAGS=-mod=

echo "=== Update Codegen for exit-handler"

rm -rf "${CODE_GEN_DIR}"

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
CODEGEN_PKG=vendor/k8s.io/code-generator
bash ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/client \
  github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/apis \
  "exithandler:v1alpha1" \
  --go-header-file ./tekton-catalog/exit-handler/hack/boilerplate/boilerplate.go.txt \
 --output-base "${CODE_GEN_DIR}"

# Knative Injection
KNATIVE_CODEGEN_PKG=vendor/knative.dev/pkg
bash ${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh "injection" \
  github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/client \
  github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/apis \
  "exithandler:v1alpha1" \
  --go-header-file ./tekton-catalog/exit-handler/hack/boilerplate/boilerplate.go.txt \
 --output-base "${CODE_GEN_DIR}"

cp -r "${CODE_GEN_DIR}/github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/client" backend/src/v2/tekton-exithandler/
cp -r "${CODE_GEN_DIR}/github.com/kubeflow/pipelines/backend/src/v2/tekton-exithandler/apis/exithandler/v1alpha1/zz_generated.deepcopy.go" backend/src/v2/tekton-exithandler/apis/exithandler/v1alpha1/

rm -rf "${CODE_GEN_DIR}"
