#!/bin/bash
#
# Copyright 2020 kubeflow.org
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

wait_for_namespace () {
    if [[ $# -ne 3 ]]
    then
        echo "Usage: wait_for_namespace namespace max_retries sleep_time"
        return 1
    fi 
    
    local namespace=$1
    local max_retries=$2
    local sleep_time=$3

    local i=0

    while [[ $i -lt $max_retries ]]
    do
        local namespaces=$(kubectl get ns)
        if [[ -n `echo ${namespaces[@]} | grep -ow $namespace` ]]
        then
            return 0
        fi
        echo "$namespace not found. Checking again in ${sleep_time}s."
        sleep $sleep_time
        i=$(($i+1))
    done

    return 1
}

wait_for_pods () {
    if [[ $# -ne 3 ]]
    then
        echo "Usage: wait_for_pods namespace max_retries sleep_time"
        return 1
    fi 

    local namespace=$1
    local max_retries=$2
    local sleep_time=$3

    local i=0

    while [[ $i -lt $max_retries ]]
    do
        local pods=$(kubectl get pod -n $namespace)

        if [[ -z $pods ]]
        then
            echo "Missing pods."
            return 1
        fi

        # Using quotations around variables to keep column format in echo
        # Remove 1st line (header line) -> trim whitespace -> cut statuses column (3rd column)
        # Might be overkill to parse down to specific columns :).
        local statuses=$(echo "$pods" | tail -n +2 | tr -s ' '  | cut -d ' ' -f 3)
        local num_pods=$(echo "$statuses" | wc -l | xargs)
        local num_running=$(echo "$statuses" | grep -ow "Running\|Completed" | wc -l | xargs)

        local msg="${num_running}/${num_pods} pods running in \"${namespace}\"."

        if [[ $num_running -ne $num_pods ]]
        then
            echo "$msg Checking again in ${sleep_time}s."
        else
            echo $msg
            return 0
        fi
        sleep $sleep_time
        i=$(($i+1))
    done

    return 1
}

deploy_with_retries () {
    if [[ $# -ne 3 ]]
    then
        echo "Usage: deploy_with_retries manifest max_retries sleep_time"
        return 1
    fi 

    local manifest=$1
    local max_retries=$2
    local sleep_time=$3

    local i=0

    while [[ $i -lt $max_retries ]]
    do
        local exit_code=0

        kubectl apply -k $manifest || exit_code=$?

        if [[ $exit_code -eq 0 ]]
        then
            return 0
        fi
        
        echo "Deploy unsuccessful with error code $exit_code. Trying again in ${sleep_time}s."
        sleep $sleep_time
        i=$(($i+1))
    done

    return 1
}
