#!/bin/bash


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

counter=0
anotherfunction() {
    echo "another function"
    if [[ "$counter" -eq 1 ]]; then
        return 0
    fi
    counter=$((counter+1))
    return 1
}
retry 3 5 anotherfunction
