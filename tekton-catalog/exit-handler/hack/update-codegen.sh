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

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT_DIR=$(dirname "${BASH_SOURCE%/*}")
CODE_GEN_DIR=code-gen

go mod vendor
source $(dirname $0)/../vendor/knative.dev/hack/codegen-library.sh

# If we run with -mod=vendor here, then generate-groups.sh looks for vendor files in the wrong place.
export GOFLAGS=-mod=

echo "=== Update Codegen for ${MODULE_NAME}"

group "Kubernetes Codegen"

rm -rf "${CODE_GEN_DIR}"

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/client \
  github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/apis \
  "exithandler:v1alpha1" \
  --go-header-file "${REPO_ROOT_DIR}/hack/boilerplate/boilerplate.go.txt" \
 --output-base "${CODE_GEN_DIR}"

group "Knative Codegen"

# Knative Injection
${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh "injection" \
  github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/client \
  github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/apis \
  "exithandler:v1alpha1" \
  --go-header-file "${REPO_ROOT_DIR}/hack/boilerplate/boilerplate.go.txt" \
 --output-base "${CODE_GEN_DIR}"

cp -r "${CODE_GEN_DIR}/github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/client" pkg/
cp -r "${CODE_GEN_DIR}/github.com/kubeflow/kfp-tekton/tekton-catalog/exit-handler/pkg/apis/exithandler/v1alpha1/zz_generated.deepcopy.go" pkg/apis/exithandler/v1alpha1/

rm -rf "${CODE_GEN_DIR}"

group "Update deps post-codegen"

# Make sure our dependencies are up-to-date
${REPO_ROOT_DIR}/hack/update-deps.sh

# drop the license folder
rm -rf third-party
