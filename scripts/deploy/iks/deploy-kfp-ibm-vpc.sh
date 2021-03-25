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
# This script should run as part of ./deploy-ibm-vpc, but can be used stand alone as well.

set -o pipefail

# cat /dev/urandom produces an exit code other than 0.
RAND_STR=$(cat -v /dev/urandom | LC_ALL=C tr -cd 'a-z0-9' | head -c 4)

set -e

# Convert a user name string to a trimmed and normalized (i.e. lower case alphanumeric string).
USER_STR=$(echo "$USER" | LC_ALL=C tr -cd 'a-z0-9' | head -c 8)

# These are reasonable defaults, actual values will be passed on from the ./deploy-ibm-vpc.sh script.
# Ideally, BASE_DIR will be the name of $HOME/VPC and KF_NAME will match the name of the cluster.

# Set the configuration file to use, such as:
export KF_CONFIG_FILE=kfctl_ibm.yaml
export KF_CONFIG_URI="https://raw.githubusercontent.com/kubeflow/manifests/v1.2-branch/kfdef/kfctl_ibm.v1.2.0.yaml"

function download_kfctl() {
    cd "$KF_INSTALL_DIR"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        curl -s -L https://github.com/kubeflow/kfctl/releases/download/v1.2.0/kfctl_v1.2.0-0-gbc038f9_darwin.tar.gz -o kfctl.tar.gz
    elif [[ "$OSTYPE" == "linux"* ]]; then
        curl -s -L https://github.com/kubeflow/kfctl/releases/download/v1.2.0/kfctl_v1.2.0-0-gbc038f9_linux.tar.gz -o kfctl.tar.gz
    else
        echo "Operating not supported. exiting..."
        exit 2
    fi
    tar -xf kfctl.tar.gz
}

function assert_gt() {
    if [[ $# -ne 3 ]]; then
        echo "Usage: assert val1 val2 msg # Where val1 is asserted to be greater than val2."
        return 1
    fi
    local val1="$1"
    local val2="$2"
    local msg="$3"

    if [[ "$val1" -gt "$val2" ]]; then
        return 0
    fi

    echo "ERROR: Assert failed:, $val1 is not greater than $val2. $msg"
    return 2
}

function usage() {
    echo -e "Deploy Kubeflow to vpc-gen2 cluster."
    echo ""
    echo "./deploy-kfp-ibm-vpc.sh"
    echo -e "\t-h --help"
    echo -e "\t--kf-dir=${KF_INSTALL_DIR} (default value :BASE_DIR/KF_NAME)"
    echo -e "\t--kf-name=${KF_INSTALL_NAME} (Suitable name for the kf delpoyment. Usually same as the name of the cluster)"
    echo -e "\t--base-dir=${BASE_KF_DIR} (Directory where the installation related files will be cached. HOME/VPC_NAME)"
    echo ""
}

while [ "$1" != "" ]; do
    PARAM=$(echo "$1" | awk -F= '{print $1}')
    VALUE=$(echo "$1" | awk -F= '{print $2}')
    case $PARAM in
        -h | --help)
            usage
            exit
            ;;
        --kf-dir)
            KF_INSTALL_DIR=$VALUE
            ;;
        --kf-name)
            KF_INSTALL_NAME=$VALUE
            ;;
        --base-dir)
            BASE_KF_DIR=$VALUE
            ;;
        *)
            echo "ERROR: unknown parameter \"$PARAM\""
            usage
            exit 1
            ;;
    esac
    shift
done
export KF_INSTALL_NAME=${KF_INSTALL_NAME:-"kf-${USER_STR}-${RAND_STR}"}

export BASE_KF_DIR=${BASE_KF_DIR:-"$HOME/kubeflow_install"}
export KF_INSTALL_DIR=${KF_INSTALL_DIR:-"${BASE_KF_DIR}/${KF_INSTALL_NAME}"}

# Generate Kubeflow:
rm -rf "${KF_INSTALL_DIR}"
mkdir -p "${KF_INSTALL_DIR}"
cd "${KF_INSTALL_DIR}"
curl -L "${KF_CONFIG_URI}" >"${KF_CONFIG_FILE}"

# Download the kfctl script, if not already.
if [[ ! -x "${KF_INSTALL_DIR}/kfctl" ]]; then
    download_kfctl
fi

"${KF_INSTALL_DIR}/kfctl" build -V -f "${KF_CONFIG_FILE}"
# Deploy Kubeflow. You can customize the KF_CONFIG_FILE if needed.
"${KF_INSTALL_DIR}/kfctl" apply -V -f "${KF_CONFIG_FILE}"

# Check if kubeflow pods appear after install.
if [[ -x $(which kubectl) ]]; then
    echo "Waiting for pods to appear !"
    sleep 60s
    value=$(kubectl -n kubeflow wait --for=condition=Ready --timeout 300s --all pods | wc -l | xargs)
    assert_gt "$value" "20" "kubeflow successful install should have more than 20 pods in \`Ready\` state."
else
    echo "kubectl command not found, exiting ..."
    exit 1
fi

echo "Open http://127.0.0.1:7080/ in a browser."
set -x
kubectl -n istio-system port-forward service/istio-ingressgateway 7080:http2 1>&2 2>/dev/null &
