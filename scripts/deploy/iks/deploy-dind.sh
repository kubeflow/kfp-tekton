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
# source: https://raw.githubusercontent.com/open-toolchain/commons/master/scripts/check_registry.sh

# Remove the x if you need no print out of each command
set -xe

# Environment variables needed by this script:
# - REGION:               cloud region (us-south as default)
# - ORG:                  target organization (dev-advo as default)
# - SPACE:                target space (dev as default)
# - IBM_CLOUD_API_KEY:    iam api key
# - KUBE_CLUSTER:         kubernetes cluster name
# - DIND_NS:              kubernetes ns for DinD deployment

REGION=${REGION:-"us-south"}
ORG=${ORG:-"dev-advo"}
SPACE=${SPACE:-"dev"}
DIND_POD_NAME="docker"

DEPLOYMENT=$(cat << EOF
apiVersion: v1
kind: Pod
metadata:
    name: ${DIND_POD_NAME}
spec:
    containers:
      - name: docker-daemon
        image: docker:20.10.2-dind
        ports:
        - containerPort: 2376
          protocol: TCP
        resources:
            requests:
                cpu: 20m
                memory: 1024Mi
        securityContext:
            privileged: true
        volumeMounts:
          - name: docker-graph-storage
            mountPath: /var/lib/docker
    volumes:
      - name: docker-graph-storage
        emptyDir: {}
EOF
)

create_kubernetes_namespace() {
  local NS=$1
  shift
  kubectl create ns "$NS"
}

check_and_create_kubernetes_namespace() {
  local NS=$1
  shift
  kubectl get ns "$NS" || create_kubernetes_namespace "$NS"
}

deploy_dind() {
  local NS=$1
  shift
  shift

  echo "${DEPLOYMENT}" | kubectl apply -n "$NS" -f -
}

check_and_deploy_dind() {
  local NS=$1
  shift

  kubectl get pod -n "$NS" "$DIND_POD_NAME" || deploy_dind "$NS" "$DEPLOYMENT"
}

check_dind_running() {
  local NS=$1
  kubectl get pod "$DIND_POD_NAME" -n "$NS"
  kubectl wait --for=condition=Ready "pod/${DIND_POD_NAME}" -n "$NS" --timeout=60s
}

ibmcloud login --apikey "${IBM_CLOUD_API_KEY}" --no-region
ibmcloud target -r "$REGION" -o "$ORG" -s "$SPACE"
ibmcloud ks cluster config -c "${KUBE_CLUSTER}"

check_and_create_kubernetes_namespace "$DIND_NS"

check_and_deploy_dind "$DIND_NS"
check_dind_running "$DIND_NS"
