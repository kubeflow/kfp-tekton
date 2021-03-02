#!/bin/bash
# 
# Copyright 2021 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euxo pipefail

IBM_CLOUD_API="${IBM_CLOUD_API:=us-south}"
IBM_CLOUD_REGION="${IBM_CLOUD_REGION:="https://cloud.ibm.com"}"
IBMCLOUD_RESOURCE_GROUP="${IBMCLOUD_RESOURCE_GROUP:="default"}"
CLUSTER_NAME="${CLUSTER_NAME:=iks-cluster}"

IBM_CLOUD_REGION=$(echo "$IBM_CLOUD_REGION" | awk -F ':' '{print $NF;}')

ibmcloud config --check-version false

ibmcloud login -a "$IBM_CLOUD_API" -r "$IBM_CLOUD_REGION" --apikey "$API_KEY"

if [ "$IBMCLOUD_RESOURCE_GROUP" ]; then
    ibmcloud target -g "$IBMCLOUD_RESOURCE_GROUP"
fi

if ibmcloud ks cluster get --cluster "$CLUSTER_NAME"; then
    ibmcloud ks cluster config --cluster $CLUSTER_NAME
else
    echo "Cluster $CLUSTER_NAME not found. Accessible clusters are:"
    ibmcloud ks clusters
    exit 1
fi
