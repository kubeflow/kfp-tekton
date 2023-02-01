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

# trusted-ai example
run_trusted_ai_example() {
  local REV=1
  local DURATION=$1
  shift
  local PIPELINE_ID
  local RUN_ID
  local KFP_COMMAND="kfp-tekton"

  kubectl create clusterrolebinding pipeline-runner-extend --clusterrole \
    cluster-admin --serviceaccount=kubeflow:pipeline-runner || true

  echo " =====   trusted-ai sample  ====="
  python3 samples/trusted-ai/trusted-ai.py
  # use kubeflow namespace
  yq eval --inplace '(.spec.params[] | select(.name=="namespace")) |=.value="kubeflow"' samples/trusted-ai/trusted-ai.yaml
  retry 3 3 $KFP_COMMAND --endpoint http://localhost:8888 pipeline upload -p e2e-trusted-ai samples/trusted-ai/trusted-ai.yaml || :
  PIPELINE_ID=$($KFP_COMMAND --endpoint http://localhost:8888  pipeline list | grep 'e2e-trusted-ai' | awk '{print $2}')
  if [[ -z "$PIPELINE_ID" ]]; then
    echo "Failed to upload pipeline"
    return "$REV"
  fi

  local RUN_NAME="e2e-trusted-ai-run-$((RANDOM%10000+1))"
  retry 3 3 $KFP_COMMAND --endpoint http://localhost:8888 run submit -e exp-e2e-trusted-ai -r "$RUN_NAME" -p "$PIPELINE_ID" || :
  RUN_ID=$($KFP_COMMAND --endpoint http://localhost:8888  run list | grep "$RUN_NAME" | awk '{print $2}')
  if [[ -z "$RUN_ID" ]]; then
    echo "Failed to submit a run for trusted-ai pipeline"
    return "$REV"
  fi

  local RUN_STATUS
  ENDTIME=$(date -ud "$DURATION minute" +%s)
  while [[ "$(date -u +%s)" -le "$ENDTIME" ]]; do
    RUN_STATUS=$($KFP_COMMAND --endpoint http://localhost:8888 run list | grep "$RUN_NAME" | awk '{print $6}')
    if [[ "$RUN_STATUS" == "Succeeded" ]]; then
      REV=0
      break;
    fi
    echo "  Status of trusted-ai run: $RUN_STATUS"
    sleep 30
  done

  if [[ "$REV" -eq 0 ]]; then
    echo " =====   trusted-ai sample PASSED ====="
  else
    echo " =====   trusted-ai sample FAILED ====="
  fi

  return "$REV"
}

RESULT=0
run_trusted_ai_example 40 || RESULT=$?

STATUS_MSG=PASSED
if [[ "$RESULT" -ne 0 ]]; then
  STATUS_MSG=FAILED
fi
