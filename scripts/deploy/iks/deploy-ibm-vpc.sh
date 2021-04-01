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


# A script to deploy kubernetes cluster to IBM Cloud IKS.

# Please see the README.md file, for the usage guide.

# shellcheck disable=2155

set -o pipefail

# cat /dev/urandom produces an exit code other than 0.
RAND_STR=$(cat -v /dev/urandom  | LC_ALL=C tr -cd 'a-z0-9' | head -c 4)

set -e

# Convert a user name string to a trimmed and normalized (i.e. lower case alphanumeric string).
USER_STR=$(echo "$USER" | LC_ALL=C tr -cd 'a-z0-9' | head -c 8)

export SUBNET_ID=${SUBNET_ID:-""}
export VPC_ID=${VPC_ID:-""}
export VPC_NAME=${VPC_NAME:-""}
export KUBERNETES_VERSION=${KUBERNETES_VERSION:-"1.18"}
export CLUSTER_ZONE=${CLUSTER_ZONE:-"us-south-3"}
export CLUSTER_NAME=${CLUSTER_NAME:-""}
export WORKER_COUNT=${WORKER_COUNT:-"2"}
export WORKER_NODE_FLAVOR=${WORKER_NODE_FLAVOR:-"bx2.4x16"}

function current_env() {
    local ENV_VARS=$(cat <<-END
export SUBNET_ID=${SUBNET_ID}
export VPC_ID=${VPC_ID}
export VPC_NAME=${VPC_NAME}
export GATEWAY_ID=${GATEWAY_ID}
export KUBERNETES_VERSION=${KUBERNETES_VERSION}
export CLUSTER_ZONE=${CLUSTER_ZONE}
export CLUSTER_NAME=${CLUSTER_NAME}
export WORKER_NODE_FLAVOR=${WORKER_NODE_FLAVOR}
export WORKER_COUNT=${WORKER_COUNT}
END
          )
    echo -e "$ENV_VARS"
}

function record_current_env() {
    touch "$CONFIG_FILE_IKS_DEPLOY"
    local ENV_VARS="$(current_env)"
    echo "$ENV_VARS">"$CONFIG_FILE_IKS_DEPLOY"
    chmod +x "$CONFIG_FILE_IKS_DEPLOY"
}

function usage() {
    echo -e "Deploy a IBM Cloud Kubernetes service (IKS) on vpc-gen2 provider and setup networking."
    echo ""
    echo "./deploy-ibm-vpc.sh"
    echo -e "\t-h --help"
    echo -e "\t--cluster-zone=${CLUSTER_ZONE} (One of \`ibmcloud ks locations --provider vpc-gen2 \`)"
    echo -e "\t--cluster-name=${CLUSTER_NAME} (required) (A cluster name must be unique across the zone.)"
    echo -e "\t--k8s-version=${KUBERNETES_VERSION} (One of \`ibmcloud ks versions --show-version=Kubernetes \`)"
    echo -e "\t--subnet-id=${SUBNET_ID} (auto created if not set)."
    echo -e "\t--vpc-name=${VPC_NAME} (required) (Should be a unique lower-cased alpha numeric value)."
    echo -e "\t--worker-count=${WORKER_COUNT} (A positive value greater than one.)"
    echo -e "\t--worker-node-flavor=${WORKER_NODE_FLAVOR}. (One of \`ibmcloud ks flavors --zone <zone>\`)"
    echo -e "\t--deploy-kubeflow=(true|false) (Whether to deploy kubeflow to the cluster. Default: true)"
    echo -e "\t--delete-cluster=(full|cluster) (pass \`full\` to also delete all the associated resources e.g. VPC, subnet, public gateway.)"
    echo -e "\t--config-file=$CONFIG_FILE_IKS_DEPLOY (Location of the file that stores configuration.)"
    echo ""
}

# existence check passes if a specified variable is in scope and contains a value else fail.
function existence_check() {
    # look up a variable name with the value contained in the passed variable i.e. double eval.
    local VALUE=$(eval echo "\$$1")

    if [[ "x$VALUE" == "x" ]]; then
        echo "Variable \`$1\` is not specified, please provide \`--$2=value\` as CLI option."
        exit 2
    fi
}

function complete_delete_cluster() {
    delete_cluster
    VPC_ID=$(vpc_name_to_vpc_id)
    if [[ "x$SUBNET_ID" != "x" || "x$GATEWAY_ID" != "x" ]]; then
        ibmcloud is subnetd -f -q "$SUBNET_ID" || true
        ibmcloud is pubgwd -f -q "$GATEWAY_ID" || true
    else
        echo "SUBNET_ID and GATEWAY_ID is not defined skipping..."
    fi
    set +e
    ibmcloud is vpcd -f -q "$VPC_ID"
    if [[ "$?" == "0" || "x$VPC_ID" == "x" ]] ; then
        echo "VPC $VPC_ID is deleted. Complete delete successful."
        rm "$CONFIG_FILE_IKS_DEPLOY"
    else
        echo "Delete incomplete, full delete can be done when vpc has no cluster/instances/subnets etc... in it."
        echo "A cluster/cloud resource can only be deleted once it's attached resources are released."
        echo "Please try again, after the cluster(s) delete is complete."
    fi
    set -e
}

function cluster_name_to_cluster_id() {
    existence_check "CLUSTER_NAME" "cluster-name"
    local value_cluster_id=$(ibmcloud ks clusters -q --provider vpc-gen2 | grep "^${CLUSTER_NAME}\s" | awk '{print $2}')
    echo "$value_cluster_id"
}

function delete_cluster() {
    existence_check "CLUSTER_NAME" "cluster-name"
    local CLUSTER_ID=$(cluster_name_to_cluster_id)
    set +e
    ibmcloud ks cluster rm -q -f --force-delete-storage -c "${CLUSTER_NAME}"

    if [[ "$?" != "0" ]] ; then
        echo "Cluster delete failed."
    else
        kubectl config delete-cluster "${CLUSTER_NAME}/$CLUSTER_ID" || true
        kubectl config delete-context "${CLUSTER_NAME}/$CLUSTER_ID" || true
    fi
    set -e
}

function vpc_name_to_vpc_id() {
    existence_check "VPC_NAME" "vpc-name"
    local value_vpc_id=$(ibmcloud is vpcs -q | grep "\s${VPC_NAME}\s" | awk '{print $1}')
    echo "$value_vpc_id"
}

function get_cluster_status() {
    existence_check "CLUSTER_NAME" "cluster-name"
    local cluster_status=$(ibmcloud ks cluster ls -q --provider vpc-gen2 | grep "${CLUSTER_NAME}\s" | awk '{print $3}')
    if [[ "x${cluster_status}" != "x" ]]; then
        echo "$cluster_status"
    else
        echo "Not yet deployed."
    fi
}

function wait_for_cluster_start() {
    existence_check "CLUSTER_NAME" "cluster-name"
    local count=40
    local cluster_status="$(get_cluster_status)"
    while [[ "x${cluster_status}" != "xnormal" && "$count" -ne 0 ]]
    do
        sleep 60
        cluster_status="$(get_cluster_status)"
        echo "Cluster ${cluster_status}"
        count=$((count - 1))
    done
}

DELETE_CLUSTER=""
DEPLOY_KF="true"
while [ "$1" != "" ]; do
    PARAM=$(echo "$1" | awk -F= '{print $1}')
    VALUE=$(echo "$1" | awk -F= '{print $2}')
    case $PARAM in
        -h | --help)
            usage
            exit
            ;;
        --cluster-zone)
            CLUSTER_ZONE=$VALUE
            ;;
        --cluster-name)
            CLUSTER_NAME=$VALUE
            ;;
        --k8s-version)
            KUBERNETES_VERSION=$VALUE
            ;;
        --subnet-id)
            SUBNET_ID=$VALUE
            ;;
        --vpc-name)
            VPC_NAME=$VALUE
            ;;
        --worker-count)
            WORKER_COUNT=$VALUE
            ;;
        --worker-node-flavor)
            WORKER_NODE_FLAVOR=$VALUE
            ;;
        --deploy-kubeflow)
            DEPLOY_KF=$VALUE
            ;;
        --config-file)
            CONFIG_FILE_IKS_DEPLOY=$VALUE
            ;;
        --delete-cluster)
            DELETE_CLUSTER=$VALUE
            ;;
        *)
            echo "ERROR: unknown parameter \"$PARAM\""
            usage
            exit 1
            ;;
    esac
    shift
done
existence_check "VPC_NAME" "vpc-name"
existence_check "CLUSTER_NAME" "cluster-name"

# To create a cluster.
## Step 1.
# 1a. Test if ibmcloud utility exists.
if [ -x "$(which ibmcloud)" ]; then
    echo "ibmcloud is already installed,"
else
    echo "installing ibmcloud utility."
    curl -sL https://ibm.biz/idt-installer | bash 1>&2 2>/dev/null
fi

echo "make sure you are logged-in by either \`ibmcloud login \`"
echo " or \` ibmcloud login --sso\`."

# 1b. Test if ibmcloud vpc-infrastructure/k8s plugin is installed and updated.
# 1>/dev/null exists so that script can run in embedded mode.
echo "installing vpc-infrastructure & kubernetes-service ibmcloud cli plugins..."
ibmcloud plugin update -f -q kubernetes-service 1>/dev/null
ibmcloud plugin install -f -q vpc-infrastructure 1>/dev/null
echo "done."

echo -e "Environment variables for cluster ..."
export CONFIG_FILE_IKS_DEPLOY=${CONFIG_FILE_IKS_DEPLOY:-"$PWD/$VPC_NAME-$CLUSTER_NAME-cluster-config.sh"}
echo -e "$(current_env)"

# Restore environment created by previous runs if any.
if [ -f "$CONFIG_FILE_IKS_DEPLOY" ]; then
    # shellcheck disable=SC1090
    source "$CONFIG_FILE_IKS_DEPLOY"
else
    echo "No previous run environment found."
fi

if [[ "x$DELETE_CLUSTER" != "x" ]]; then
    if [[ "x${DELETE_CLUSTER}" == "xfull" ]]; then
        complete_delete_cluster
    else
        delete_cluster
    fi
    exit 1
fi
echo "Please note the config file name, $CONFIG_FILE_IKS_DEPLOY, it can be used to perform maintenance tasks."
echo "e.g. to completely delete VPC: ./deploy-ibm-vpc.sh --config-file=$CONFIG_FILE_IKS_DEPLOY --delete-cluster=full"
echo " or just delete the cluster: ./deploy-ibm-vpc.sh --config-file=$CONFIG_FILE_IKS_DEPLOY --delete-cluster=cluster"
echo " or start a cluster in same vpc: ./deploy-ibm-vpc.sh --config-file=$CONFIG_FILE_IKS_DEPLOY --cluster-name=cluster2"
## Step 2.

# 2 Switch target to gen 2 services.
## At the moment, gen1 vpc is deprecated. So we need to explicitly set gen2 as target.

ibmcloud is target --gen 2 1>/dev/null

# 3. Select VPC.

VPC_ID=$(vpc_name_to_vpc_id)

if [[ "x${VPC_ID}" != "x" ]]; then
    echo "Using an existing VPC with vpc-name: $VPC_NAME and ID: $VPC_ID"
else
    echo "Creating a new vpc with name: $VPC_NAME"
    VPC_ID=$(ibmcloud is vpc-create -q "$VPC_NAME" |grep "^ID\b\s" | awk '{print $2}')
    echo "Created VPC: $VPC_NAME with ID: $VPC_ID"
    # When we create a new VPC, we remove old SUBNET_ID and GATEWAY_IDs
    unset "SUBNET_ID"
    unset "GATEWAY_ID"
fi
# A VPC is created recording the current env state.
record_current_env

#4. Subnet and attach a public gateway in the same step.
# If cluster already exists, then we should try to get the subnet ID it is on.

CLUSTER_ID=$(cluster_name_to_cluster_id)
if [[ "x$CLUSTER_ID" != "x" ]]; then
    echo "A Cluster with name: $CLUSTER_NAME and ID: $CLUSTER_ID, is already running. Not creating."
    CREATE_CLUSTER="false"
fi

if [[ "x${SUBNET_ID}" != "x" ]]; then
    echo "Using an existing SUBNET: $SUBNET_ID for VPC with vpc-id: ${VPC_ID}."
else
    if [[ "x${CREATE_CLUSTER}" != "xfalse" ]]; then
        SUBNET_NAME="kf-${USER_STR}-${RAND_STR}-sub"
        SUBNET_ID=$(ibmcloud is subnet-create "$SUBNET_NAME" "$VPC_ID" --ipv4-address-count 16 --zone "$CLUSTER_ZONE" -q | grep "^ID\b\s" | awk '{print $2}')
        echo "Created subnet with name: $SUBNET_NAME and ID: $SUBNET_ID."
        GATEWAY_ID=$(ibmcloud is public-gateway-create -q "kf-${USER_STR}-${RAND_STR}-gw" "$VPC_ID" "$CLUSTER_ZONE" | grep "^ID\b\s" | awk '{print $2}')
        ibmcloud is subnet-update "$SUBNET_ID" --public-gateway-id "$GATEWAY_ID" 1>/dev/null
        echo "created public gateway: $GATEWAY_ID and attached it the subnet: $SUBNET_ID"
    fi
fi
# A SUBNET is created recording the current env state.
record_current_env

#5. Create cluster.

if [[ "x$CREATE_CLUSTER" != "xfalse" ]]; then
    echo "Creating cluster with name: $CLUSTER_NAME"

    ibmcloud ks cluster create vpc-gen2 \
             --name "$CLUSTER_NAME" \
             --zone "$CLUSTER_ZONE" \
             --version "${KUBERNETES_VERSION}" \
             --flavor "${WORKER_NODE_FLAVOR}" \
             --vpc-id "${VPC_ID}" \
             --subnet-id "${SUBNET_ID}" \
             --workers "${WORKER_COUNT}"

    echo "Cluster usually takes about 15 to 30 minutes to deploy."
fi
wait_for_cluster_start

#6. Update the kubectl context.

ibmcloud ks cluster config --cluster "${CLUSTER_NAME}"

echo "The cluster: $CLUSTER_NAME in zone: $CLUSTER_ZONE, is ready."

if [ -x "$(which kubectl)" ]; then
    kubectl wait --for=condition=Ready --timeout=200s --all nodes
fi

if [[ "x$DEPLOY_KF" == "xtrue" ]]; then
  ./deploy-kfp-ibm-vpc.sh --kf-name="$CLUSTER_NAME" --base-dir="$PWD/$VPC_NAME"
fi
