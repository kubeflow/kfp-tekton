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
export CONFIG_FILE_IKS_DEPLOY=${CONFIG_FILE_IKS_DEPLOY:-"$HOME/.iks-cluster-config-env.sh"}
export WORKER_NODE_FLAVOR=${WORKER_NODE_FLAVOR:-"bx2.4x16"}

# Restore environment created by previous runs if any.
if [ -f "$CONFIG_FILE_IKS_DEPLOY" ]; then
    # shellcheck disable=SC1091
    source "$CONFIG_FILE_IKS_DEPLOY"
else
    echo "No previous run environment found."
fi

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
    echo -e "\t--cluster-name=${CLUSTER_NAME} (A cluster name must be unique across the zone.)"
    echo -e "\t--k8s-version=${KUBERNETES_VERSION} (One of \`ibmcloud ks versions --show-version=Kubernetes \`)"
    echo -e "\t--subnet-id=${SUBNET_ID} (auto created if not set)."
    echo -e "\t--vpc-name=${VPC_NAME} (auto created if not set, should be a lower-cased alpha numeric value)."
    echo -e "\t--worker-count=${WORKER_COUNT} (A positive value greater than one.)"
    echo -e "\t--worker-node-flavor=${WORKER_NODE_FLAVOR}. (One of \`ibmcloud ks flavors --zone <zone>\`)"
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
    GATEWAY_ID=$(vpc_name_to_gateway_id)
    SUBNET_ID=$(vpc_name_to_subnet_ids)
    ibmcloud is subnetd -f -q "$SUBNET_ID" || true
    ibmcloud is pubgwd -f -q "$GATEWAY_ID" || true
    set +e
    ibmcloud is vpcd -f -q $VPC_ID
    if [[ "$?" == "0" || "x$VPC_ID" == "x" ]] ; then
        echo "Complete delete successful."
        rm "$CONFIG_FILE_IKS_DEPLOY"
    else
        echo "Delete incomplete, this happens because, it takes a while for the cluster to delete(~45mins)."
        echo "A cluster/cloud resource can only be deleted once it's attached resources are released."
        echo "And cluster has \`vnic\` attahced to the subnet which is attached to a VPC."
        echo "So, to fully delete the VPC along with all it's attached resources."
        echo "Please try again, after the cluster delete is complete."
    fi
    set -e
}

function vpc_name_to_gateway_id() {
  # A VPC may have more than one Gateway attached to it's subnets. So this is not the correct way to detect.
    existence_check "VPC_NAME" "vpc-name"
    local value_pubgw_id=$(ibmcloud is pubgws -q | grep "\s${VPC_NAME}\s" | awk '{print $1}')
    echo "$value_pubgw_id"
}

function subnet_id_to_gateway_id() {
    existence_check "SUBNET_ID" "subnet-id"
    local pubgw_name=$(ibmcloud is subnets -q | grep "${SUBNET_ID}\s" | awk '{print $7}')
    if [[ "x$pubgw_name" == "x-" ]]; then
      echo ""
    else
      local value_pubgw_id=$(ibmcloud is pubgws -q | grep "\s${pubgw_name}\s" | awk '{print $1}')
      echo "$value_pubgw_id"
    fi
}

function cluster_name_to_cluster_id() {
    existence_check "CLUSTER_NAME" "cluster-name"
    local value_cluster_id=$(ibmcloud ks clusters -q --provider vpc-gen2 | grep "^${CLUSTER_NAME}\s" | awk '{print $2}')
    echo "$value_cluster_id"
}

function vpc_name_to_subnet_ids_without_pg() {
    # Current APIS do not have way to determine SUBNET ID of a vpc-gen2 cluster.
    # One way is to determine all the IPs of the nodes in the cluster and then match each subnets' CIDR block
    # to see if the subnet may contain that IP. Still it can be challenging if two different subnets exist in two
    # two different zones and have same vpc-prefixes (not sure if IBM Cloud allows this.).
    # But for our purposes, we just list all subnets in a particular VPC and see if it has a public gateway.
    existence_check "VPC_NAME" "vpc-name"
    local subnets_of_vpc_without_pg=$(ibmcloud is subnets -q | grep "\s${VPC_NAME}\s" | awk '{ if ($7 == "-") print $1 }' )
    echo "$subnets_of_vpc_without_pg"
}

function delete_cluster() {
    existence_check "CLUSTER_NAME" "cluster-name"
    local CLUSTER_ID=$(cluster_name_to_cluster_id)
    ibmcloud ks cluster rm -q -f --force-delete-storage -c "${CLUSTER_NAME}" || true
    kubectl config delete-cluster "${CLUSTER_NAME}/$CLUSTER_ID" || true
    kubectl config delete-context "${CLUSTER_NAME}/$CLUSTER_ID" || true
}

function vpc_name_to_vpc_id() {
    existence_check "VPC_NAME" "vpc-name"
    local value_vpc_id=$(ibmcloud is vpcs -q | grep "\s${VPC_NAME}\s" | awk '{print $1}')
    echo "$value_vpc_id"
}

function vpc_name_to_subnet_ids() {
    existence_check "VPC_NAME" "vpc-name"
    local value_subnet_id=$(ibmcloud is subnets -q | grep "\s${VPC_NAME}\s" | awk '{print $1}')
    echo "$value_subnet_id"
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

echo -e "Environment variables for cluster ..."
echo -e "$(current_env)"
record_current_env

if [[ "x$DELETE_CLUSTER" != "x" ]]; then
    if [[ "x${DELETE_CLUSTER}" == "xfull" ]]; then
        complete_delete_cluster
    else
        delete_cluster
    fi
    exit 1
fi

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
## Step 2.

# 2 Switch target to gen 2 services.
## At the moment, gen1 vpc is deprecated. So we need to explicitly set gen2 as target.

ibmcloud is target --gen 2 1>/dev/null

# 3. Select VPC.

if [[ "x${VPC_NAME}" != "x" ]]; then
    VPC_ID=$(vpc_name_to_vpc_id)
fi

if [[ "x${VPC_ID}" != "x" ]]; then
    echo "Using an existing VPC with vpc-name: $VPC_NAME and ID: $VPC_ID"
else
    if [[ "x${VPC_NAME}" == "x" ]]; then
        VPC_NAME="kf-${USER_STR}-${RAND_STR}-vpc"
    fi
    echo "Creating a new vpc with name: $VPC_NAME"
    VPC_ID=$(ibmcloud is vpc-create -q "$VPC_NAME" |grep "^ID\b\s" | awk '{print $2}')
    echo "Created VPC: $VPC_NAME with ID: $VPC_ID"
    # When we create a new VPC, we remove old SUBNET_ID and GATEWAY_IDs
    unset "SUBNET_ID"
    unset "GATEWAY_ID"
fi
# A VPC is created recording the current env state.
record_current_env

#4. Subnet
# If cluster already exists, then we should try to get the subnet ID it is on.
if [[ "x$CLUSTER_NAME" != "x" ]]; then
    CLUSTER_ID=$(cluster_name_to_cluster_id)
    if [[ "x$CLUSTER_ID" != "x" ]]; then
        echo "A Cluster with name: $CLUSTER_NAME and ID: $CLUSTER_ID, is already running. Not creating."
        CREATE_CLUSTER="false"
    fi
else
    CLUSTER_NAME="kf-${USER_STR}-${RAND_STR}"
fi

if [[ "x${SUBNET_ID}" != "x" ]]; then
    echo "Using an existing SUBNET: $SUBNET_ID for VPC with vpc-id: ${VPC_ID}."
else
    if [[ "x${CREATE_CLUSTER}" == "xfalse" ]]; then
        if [[ "x${GATEWAY_ID}" == "x" ]]; then
            SUBNET_ID=$(vpc_name_to_subnet_ids_without_pg)
        else
            echo "ERROR: wrong config detected, a GATEWAY_ID is set to $GATEWAY_ID, but SUBNET_ID:$SUBNET_ID is unset."
            exit 1
        fi
    else
        SUBNET_NAME="kf-${USER_STR}-${RAND_STR}-sub"
        SUBNET_ID=$(ibmcloud is subnet-create "$SUBNET_NAME" "$VPC_ID" --ipv4-address-count 16 --zone "$CLUSTER_ZONE" -q | grep "^ID\b\s" | awk '{print $2}')
        echo "Created subnet with name: $SUBNET_NAME and ID: $SUBNET_ID."
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

#6. Attach a public gateway.
# Check if the public gateway is already attached to VPC,
# At this point SUBNET_ID must be defined, else script will fail with proper error.
GATEWAY_ID="$(subnet_id_to_gateway_id)"

if [[ "x${GATEWAY_ID}" != "x" ]]; then
    echo "Found a public gateway already attached to VPC: ${VPC_NAME} with gateway id: ${GATEWAY_ID}."
else
    GATEWAY_ID=$(ibmcloud is public-gateway-create -q "kf-${USER_STR}-${RAND_STR}-gw" "$VPC_ID" "$CLUSTER_ZONE" | grep "^ID\b\s" | awk '{print $2}')
    ibmcloud is subnet-update "$SUBNET_ID" --public-gateway-id "$GATEWAY_ID" 1>/dev/null
    echo "created public gateway: $GATEWAY_ID and attached it the subnet: $SUBNET_ID"
fi
record_current_env
#7. Update the kubectl context.

ibmcloud ks cluster config --cluster "${CLUSTER_NAME}"

echo "The cluster: $CLUSTER_NAME in zone: $CLUSTER_ZONE, is ready."

if [ -x "$(which kubectl)" ]; then
    kubectl wait --for=condition=Ready --timeout=200s --all nodes
fi

./deploy-kfp-ibm-vpc.sh --kf-name="$CLUSTER_NAME" --base-dir="$HOME/$VPC_NAME"

# Test cases:
# 1. ./deploy-ibm-vpc.sh
#
# value=$(kubectl get nodes | wc -l | xargs)
# assert [[ value == 2 ]]
#
# 2. ./deploy-ibm-vpc.sh --delete-cluster=full
# sleep 3600s
# value=$(kubectl get nodes)
# assert [[ value == "" ]]
# ./deploy-ibm-vpc.sh --delete-cluster=full
#
# assert [[ ibmcloud is vpcs -q | wc -l  -eq 0 ]]
# 3. ./deploy-ibm-vpc.sh --vpc-name="xyz-test" --cluster-name="abc-test"
#
# value=$(ibmcloud ks clusters -q | grep "abc-test" | wc -l | xargs)
# assert [[ value -eq "1" ]]
#
