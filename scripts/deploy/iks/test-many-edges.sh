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

# a pipeline with many edges which triggers validation in webhook
run_many_edges() {
  local REV=1
  local DURATION=$1
  shift
  local PIPELINE_ID
  local RUN_ID

  echo " =====  many edges  ====="
  python3 scripts/deploy/iks/test/many-edges.py
  retry 3 3 kfp --endpoint http://localhost:8888 pipeline upload -p many-edges scripts/deploy/iks/test/many-edges.yaml || :
  PIPELINE_ID=$(kfp --endpoint http://localhost:8888  pipeline list | grep 'many-edges' | awk '{print $2}')
  if [[ -z "$PIPELINE_ID" ]]; then
    echo "Failed to upload pipeline"
    return "$REV"
  fi

  local RUN_NAME="many-edges-run-$((RANDOM%10000+1))"
  local ENDTIME

  ENDTIME=$(date -ud "5 second" +%s)
  retry 3 3 kfp --endpoint http://localhost:8888 run submit -e exp-many-edges -r "$RUN_NAME" -p "$PIPELINE_ID" || :
  RUN_ID=$(kfp --endpoint http://localhost:8888  run list | grep "$RUN_NAME" | awk '{print $2}')
  if [[ -z "$RUN_ID" ]]; then
    echo "Failed to submit a run for many edges pipeline"
    return "$REV"
  fi

  if [[ "$(date -u +%s)" -gt "$ENDTIME" ]]; then
    echo "validation duration is longer then 5 seconds"
    return "$REV"
  fi

  # only need to verify the webhook validation. no need to verify the run status
  echo " =====   many edges PASSED ====="

  return 0
}

RESULT=0
run_many_edges 20 || RESULT=$?

STATUS_MSG=PASSED
if [[ "$RESULT" -ne 0 ]]; then
  STATUS_MSG=FAILED
fi