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


# Install kubeflow with all the common operators and components on IBM Cloud IKS vpc-gen2 cluster.
set -o pipefail

# cat /dev/urandom produces an exit code other than 0.
RAND_STR=$(cat -v /dev/urandom  | LC_ALL=C tr -cd 'a-z0-9' | head -c 4)

set -e

# Convert a user name string to a trimmed and normalized (i.e. lower case alphanumeric string).
USER_STR=$(echo $USER | LC_ALL=C tr -cd 'a-z0-9' | head -c 8)

# These are reasonable defaults, these values will be passed on from the ./deploy-ibm-vpc.sh script.
# Ideally, BASE_DIR will be the name of VPC and KF_NAME will match the name of the cluster.
export KF_NAME=${KF_NAME:-"kf-${USER_STR}-${RAND_STR}"}

export BASE_DIR=${BASE_DIR:-"$HOME"}
export KF_DIR=${KF_DIR:-"${BASE_DIR}/${KF_NAME}"}

# Set the configuration file to use, such as:
export CONFIG_FILE=kfctl_ibm.yaml
export CONFIG_URI="https://raw.githubusercontent.com/kubeflow/manifests/v1.2-branch/kfdef/kfctl_ibm.v1.2.0.yaml"


function usage() {
    echo -e "Deploy Kubeflow to vpc-gen2 cluster."
    echo ""
    echo "./deploy-kfp-ibm-vpc.sh"
    echo -e "\t-h --help"
    echo -e "\t--kf-dir=${KF_DIR}"
    echo -e "\t--kf-name=${CLUSTER_NAME} (A cluster name must be unique across the zone.)"
    echo -e "\t--k8s-version=${KUBERNETES_VERSION} (default)"
    echo -e "\t--subnet-id=${SUBNET_ID} (auto created if not set)."
    echo -e "\t--vpc-name=${VPC_NAME} (auto created if not set)."
    echo -e "\t--worker-count=${WORKER_COUNT}."
    echo -e "\t--worker-node-flavor=${WORKER_NODE_FLAVOR}."
    echo -e "\t--delete-cluster=(full|cluster) pass \`full\` to also delete all the associated resources e.g. VPC, subnet, public gateway."
    echo -e "\t--config-file=$CONFIG_FILE_IKS_DEPLOY (default) Location of the file that stores configuration."
    echo ""
}


# Generate Kubeflow:
mkdir -p ${KF_DIR}
cd ${KF_DIR}
curl -L ${CONFIG_URI} > ${CONFIG_FILE}

# Deploy Kubeflow. You can customize the CONFIG_FILE if needed.
kfctl apply -V -f ${CONFIG_FILE}