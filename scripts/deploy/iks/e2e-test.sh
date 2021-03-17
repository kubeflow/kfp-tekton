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

# Remove the x if you need no print out of each command
set -ex

# Need the following env
# - PIPELINE_KUBERNETES_CLUSTER_NAME:       kube cluster name
# - KUBEFLOW_NS:                            kubeflow namespace
# - SLACK_WEBHOOK:                          webhook to send out the notification
# - SLACK_CHANNEL:                          slack channel name
# - PIPELINE_URL:                           url to point of the details page for this pipeline run

SLACK_WEBHOOK="${SLACK_WEBHOOK:-""}"
SLACK_CHANNEL="${SLACK_CHANNEL:-""}"
PIPELINE_URL="${PIPELINE_URL:-""}"

# These env vars should come from the build.properties that `build-image.sh` generates
echo "REGISTRY_URL=${REGISTRY_URL}"
echo "REGISTRY_NAMESPACE=${REGISTRY_NAMESPACE}"
echo "BUILD_NUMBER=${BUILD_NUMBER}"
echo "ARCHIVE_DIR=${ARCHIVE_DIR}"
echo "GIT_BRANCH=${GIT_BRANCH}"
echo "GIT_COMMIT=${GIT_COMMIT}"
echo "GIT_COMMIT_SHORT=${GIT_COMMIT_SHORT}"
echo "REGION=${REGION}"
echo "ORG=${ORG}"
echo "SPACE=${SPACE}"
echo "RESOURCE_GROUP=${RESOURCE_GROUP}"
echo "PIPELINE_KUBERNETES_CLUSTER_NAME=${PIPELINE_KUBERNETES_CLUSTER_NAME}"
echo "KUBEFLOW_NS=${KUBEFLOW_NS}"
echo "PIPELINE_URL=${PIPELINE_URL}"
echo "SLACK_CHANNEL=${SLACK_CHANNEL}"

# copy files to ARCHIVE_DIR for next stage if needed
echo "Checking archive dir presence"
if [[ -z "$ARCHIVE_DIR" || "$ARCHIVE_DIR" == "." ]]; then
  echo -e "Build archive directory contains entire working directory."
else
  echo -e "Copying working dir into build archive directory: ${ARCHIVE_DIR} "
  mkdir -p "$ARCHIVE_DIR"
  find . -mindepth 1 -maxdepth 1 -not -path "./${ARCHIVE_DIR}" -exec cp -R '{}' "${ARCHIVE_DIR}/" ';'
fi
cp build.properties "${ARCHIVE_DIR}/" || :

retry() {
  local max=$1; shift
  local interval=$1; shift

  until "$@"; do
    echo "trying.."
    max=$((max-1))
    if [[ "$max" -eq 0 ]]; then
      return 1
    fi
    sleep "$interval"
  done
}

check_kfp_pipeline() {
  kubectl get pod -n "$KUBEFLOW_NS"
  until kubectl get pod -l app=ml-pipeline -n "$KUBEFLOW_NS" | grep -q  '1/1'; do
    sleep 10; echo 'wait for 10s';
  done
}

# Set up kubernetes config
retry 3 3 ibmcloud login --apikey "${IBM_CLOUD_API_KEY}" --no-region
retry 3 3 ibmcloud target -r "$REGION" -o "$ORG" -s "$SPACE" -g "$RESOURCE_GROUP"
retry 3 3 ibmcloud ks cluster config -c "$PIPELINE_KUBERNETES_CLUSTER_NAME"

# make sure ml-pipeline is up and running
check_kfp_pipeline

POD_NAME=$(kubectl get pod -n kubeflow -l app=ml-pipeline -o json | jq -r '.items[] | .metadata.name ')
kubectl port-forward -n "$KUBEFLOW_NS" "$POD_NAME" 8888:8888 &
# wait for the port-forward
sleep 5

# Prepare python venv and install sdk
python3 -m venv .venv
source .venv/bin/activate
pip install wheel
pip install -e sdk/python
pip install -U setuptools
pip install pytest

# flip coin example
run_flip_coin_example() {
  local REV=1
  local DURATION=$1
  shift
  local PIPELINE_ID
  local RUN_ID

  echo " =====   flip coin sample  ====="
  python3 samples/flip-coin/condition.py
  retry 3 3 kfp --endpoint http://localhost:8888 pipeline upload -p e2e-flip-coin samples/flip-coin/condition.yaml || :
  PIPELINE_ID=$(kfp --endpoint http://localhost:8888  pipeline list | grep 'e2e-flip-coin' | awk '{print $2}')
  if [[ -z "$PIPELINE_ID" ]]; then
    echo "Failed to upload pipeline"
    return "$REV"
  fi

  local RUN_NAME="e2e-flip-coin-run-$((RANDOM%10000+1))"
  retry 3 3 kfp --endpoint http://localhost:8888 run submit -e exp-e2e-flip-coin -r "$RUN_NAME" -p "$PIPELINE_ID" || :
  RUN_ID=$(kfp --endpoint http://localhost:8888  run list | grep "$RUN_NAME" | awk '{print $2}')
  if [[ -z "$RUN_ID" ]]; then
    echo "Failed to submit a run for flip coin pipeline"
    return "$REV"
  fi

  local RUN_STATUS
  ENDTIME=$(date -ud "$DURATION minute" +%s)
  while [[ "$(date -u +%s)" -le "$ENDTIME" ]]; do
    RUN_STATUS=$(kfp --endpoint http://localhost:8888 run list | grep "$RUN_NAME" | awk '{print $6}')
    if [[ "$RUN_STATUS" == "Completed" ]]; then
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
run_flip_coin_example 10 || RESULT=$?

PUBLISH_BUILD_STATUS=pass
STATUS_MSG=PASSED
if [[ "$RESULT" -ne 0 ]]; then
  PUBLISH_BUILD_STATUS=fail
  STATUS_MSG=FAILED
fi
# check if doi is integrated in this toolchain
if [ -e _toolchain.json ]; then
  if jq -e '.services[] | select(.service_id=="draservicebroker")' _toolchain.json; then
    ibmcloud doi publishbuildrecord --branch "${GIT_BRANCH}" --repositoryurl "${GIT_URL}" --commitid "${GIT_COMMIT}" \
      --buildnumber "${BUILD_NUMBER}" --logicalappname "kfp-tekton" --status "$PUBLISH_BUILD_STATUS"
  fi
fi

if [[ "$SLACK_CHANNEL" != "" && "$SLACK_WEBHOOK" != "" ]]; then
  curl -X POST --data-urlencode "payload={\"channel\": \"#${SLACK_CHANNEL}\", \"username\": \"tekton-pipeline\", \"type\": \"mrkdwn\", \"text\": \"build: *<${PIPELINE_URL} | ${BUILD_NUMBER}>* ${STATUS_MSG}.\", \"icon_emoji\": \":robot_:\"}" \
    "$SLACK_WEBHOOK"
fi

kill %1

if [[ "$RESULT" -ne 0 ]]; then
  echo "e2e test ${STATUS_MSG}"
  exit 1
fi

echo "e2e test ${STATUS_MSG}"
