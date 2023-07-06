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

# flip coin example
run_flip_coin_example() {
  local REV=1
  local DURATION=$1
  shift
  local PIPELINE_ID
  local RUN_ID
  local KFP_COMMAND="kfp"
  local PIPELINE_NAME="e2e-flip-coin-$((RANDOM%10000+1))"

  echo " =====   flip coin sample  ====="
  kfp dsl compile --py samples/core/condition/condition_v2.py --output samples/core/condition/condition_v2.yaml
  retry 3 3 $KFP_COMMAND --endpoint http://localhost:8888 pipeline create -p "$PIPELINE_NAME" samples/core/condition/condition_v2.yaml 2>&1 || :
  PIPELINE_ID=$($KFP_COMMAND --endpoint http://localhost:8888 pipeline list 2>&1| grep "$PIPELINE_NAME" | awk '{print $1}')
  if [[ -z "$PIPELINE_ID" ]]; then
    echo "Failed to upload pipeline"
    return "$REV"
  fi
  VERSION_ID=$($KFP_COMMAND --endpoint http://localhost:8888 pipeline list-version "${PIPELINE_ID}" 2>&1| grep "$PIPELINE_NAME" | awk '{print $1}')

  local RUN_NAME="${PIPELINE_NAME}-run"
  retry 3 3 $KFP_COMMAND --endpoint http://localhost:8888 run create -e exp-e2e-flip-coin -r "$RUN_NAME" -p "$PIPELINE_ID" -v "$VERSION_ID" 2>&1 || :
  RUN_ID=$($KFP_COMMAND --endpoint http://localhost:8888  run list 2>&1| grep "$RUN_NAME" | awk '{print $1}')
  if [[ -z "$RUN_ID" ]]; then
    echo "Failed to submit a run for flip coin pipeline"
    return "$REV"
  fi

  local RUN_STATUS
  ENDTIME=$(date -ud "$DURATION minute" +%s)
  while [[ "$(date -u +%s)" -le "$ENDTIME" ]]; do
    RUN_STATUS=$($KFP_COMMAND --endpoint http://localhost:8888 run list 2>&1| grep "$RUN_NAME" | awk '{print $4}')
    if [[ "$RUN_STATUS" == "SUCCEEDED" ]]; then
      REV=0
      break;
    fi
    echo "  Status of flip coin run: $RUN_STATUS"
    sleep 10
  done

  if [[ "$REV" -eq 0 ]]; then
    echo " =====   flip coin sample PASSED ====="
  else
    echo " =====   flip coin sample FAILED ====="
  fi

  return "$REV"
}

RESULT=0
run_flip_coin_example 20 || RESULT=$?

STATUS_MSG=PASSED
if [[ "$RESULT" -ne 0 ]]; then
  STATUS_MSG=FAILED
fi
