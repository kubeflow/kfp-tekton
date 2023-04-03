#!/bin/bash
#
# Copyright 2022 kubeflow.org
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

# condition depend pipeline
run_cond_dep() {
  local REV=1
  local DURATION=$1
  shift
  local PIPELINE_ID
  local RUN_ID
  local PIPELINE_NAME="cond-dep-$((RANDOM%10000+1))"
  local KFP_COMMAND="kfp-tekton"

  echo " =====   condition depend pipeline  ====="
  python3 sdk/python/tests/compiler/testdata/condition_depend.py
  retry 3 3 $KFP_COMMAND --endpoint http://localhost:8888 pipeline upload -p "$PIPELINE_NAME" sdk/python/tests/compiler/testdata/condition_depend.yaml || :
  PIPELINE_ID=$($KFP_COMMAND --endpoint http://localhost:8888  pipeline list | grep "$PIPELINE_NAME" | awk '{print $2}')
  if [[ -z "$PIPELINE_ID" ]]; then
    echo "Failed to upload pipeline"
    return "$REV"
  fi

  local RUN_NAME="${PIPELINE_NAME}-run"
  retry 3 3 $KFP_COMMAND --endpoint http://localhost:8888 run submit -e "exp-cond-dep" -r "$RUN_NAME" -p "$PIPELINE_ID" || :
  RUN_ID=$($KFP_COMMAND --endpoint http://localhost:8888  run list | grep "$RUN_NAME" | awk '{print $2}')
  if [[ -z "$RUN_ID" ]]; then
    echo "Failed to submit a run for condition-depend pipeline"
    return "$REV"
  fi

  local RUN_STATUS
  ENDTIME=$(date -ud "$DURATION minute" +%s)
  while [[ "$(date -u +%s)" -le "$ENDTIME" ]]; do
    RUN_STATUS=$($KFP_COMMAND --endpoint http://localhost:8888 run list | grep "$RUN_NAME" | awk '{print $6}')
    if [[ "$RUN_STATUS" == "Completed" ]]; then
      REV=0
      break;
    fi
    echo "  Status of condition-depend run: $RUN_STATUS"
    sleep 10
  done

  if [[ "$REV" -eq 0 ]]; then
    python3 sdk/python/tests/verify_result.py "$RUN_ID" scripts/deploy/iks/expect-condition-depend.json || REV=$?
    if [[ "$REV" -eq 0 ]]; then
      echo " =====   condition depend pipeline PASSED ====="
      return "$REV"
    fi
  fi

  echo " =====   condition depend pipeline FAILED ====="
  return "$REV"
}

RESULT=0
run_cond_dep 20 || RESULT=$?

STATUS_MSG=PASSED
if [[ "$RESULT" -ne 0 ]]; then
  STATUS_MSG=FAILED
fi