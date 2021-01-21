#!/usr/bin/env bash

# Copyright 2020 The Knative Authors
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

# This script calls out to scripts in tektoncd/plumbing to setup a cluster
# and deploy Tekton Pipelines to it for running integration tests.

source $(git rev-parse --show-toplevel)/task-loops/test/e2e-common.sh
source $(git rev-parse --show-toplevel)/task-loops/test/e2e-pipelineloops.sh

# Script entry point.

initialize $@

# initialize function does a CD to REPO_ROOT_DIR so we have to CD back here.
cd ${REPO_ROOT_DIR}/task-loops

header "Setting up environment"

install_pipeline_crd_version v0.18.0

install_pipelineloop_crd

failed=0

# Run the integration tests
header "Running Go e2e tests"
go_test_e2e -timeout=20m ./test/... || failed=1

(( failed )) && fail_test
success
