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
# - IMAGE_NAME:           image name
# - DOCKER_ROOT:          docker root
# - DOCKER_FILE:          docker file
# - REGISTRY_URL:         container registry url
# - REGISTRY_NAMESPACE:   namespace for the image
# - RUN_TASK:             execution task:
#                           - `artifact`: prepare the artifact for next stage
#                           - `image`: prune, build and push the image
#
# The full image url would be:
# ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/${IMAGE_NAME}:${BUILD_NUMBER}-${GIT_COMMIT_SHORT}

# The following envs could be loaded from `build.properties` that
# `run-test.sh` generates.
# - REGION:               cloud region (us-south as default)
# - ORG:                  target organization (dev-advo as default)
# - SPACE:                target space (dev as default)
# - GIT_BRANCH:           git branch
# - GIT_COMMIT:           git commit hash
# - GIT_COMMIT_SHORT:     git commit hash short

REGION=${REGION:-"us-south"}
ORG=${ORG:-"dev-advo"}
SPACE=${SPACE:-"dev"}
IMAGE_TAG="${BUILD_NUMBER}-${GIT_COMMIT_SHORT}"
RUN_TASK=${RUN_TASK:-"artifact"}
BUILD_ARG_LIST=${BUILD_ARG_LIST:-""}

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

retry 3 3 ibmcloud login --apikey "${IBM_CLOUD_API_KEY}" --no-region
retry 3 3 ibmcloud target -r "$REGION" -o "$ORG" -s "$SPACE" -g "$RESOURCE_GROUP"

print_env_vars() {
  echo "REGISTRY_URL=${REGISTRY_URL}"
  echo "REGISTRY_NAMESPACE=${REGISTRY_NAMESPACE}"
  echo "IMAGE_TAG=${IMAGE_TAG}"
  echo "DOCKER_ROOT=${DOCKER_ROOT}"
  echo "DOCKER_FILE=${DOCKER_FILE}"

  # These env vars should come from the build.properties that `run-test.sh` generates
  echo "BUILD_NUMBER=${BUILD_NUMBER}"
  echo "ARCHIVE_DIR=${ARCHIVE_DIR}"
  echo "GIT_BRANCH=${GIT_BRANCH}"
  echo "GIT_COMMIT=${GIT_COMMIT}"
  echo "GIT_COMMIT_SHORT=${GIT_COMMIT_SHORT}"
  echo "REGION=${REGION}"
  echo "ORG=${ORG}"
  echo "SPACE=${SPACE}"
  echo "RESOURCE_GROUP=${RESOURCE_GROUP}"
  echo "BUILD_ARG_LIST=${BUILD_ARG_LIST}"

  # View build properties
  if [ -f build.properties ]; then
    echo "build.properties:"
    grep -v -i password build.properties
  else
    echo "build.properties : not found"
  fi
}

######################################################################################
# Copy any artifacts that will be needed for subsequent stages    #
######################################################################################
artificat_for_next_stage() {
  echo "=========================================================="
  echo "Copy and prepare artificates for subsequent stages"

  echo "Checking archive dir presence"
  if [[ -z "$ARCHIVE_DIR" || "$ARCHIVE_DIR" == "." ]]; then
    echo -e "Build archive directory contains entire working directory."
  else
    echo -e "Copying working dir into build archive directory: ${ARCHIVE_DIR} "
    mkdir -p "$ARCHIVE_DIR"
    find . -mindepth 1 -maxdepth 1 -not -path "./${ARCHIVE_DIR}" -exec cp -R '{}' "${ARCHIVE_DIR}/" ';'
  fi

  # Persist env variables into a properties file (build.properties) so that all pipeline stages consuming this
  # build as input and configured with an environment properties file valued 'build.properties'
  # will be able to reuse the env variables in their job shell scripts.

  # If already defined build.properties from prior build job, append to it.
  cp build.properties "${ARCHIVE_DIR}/" || :

  # IMAGE information from build.properties is used in Helm Chart deployment to set the release name
  {
    echo "IMAGE_TAG=${IMAGE_TAG}"
    # REGISTRY information from build.properties is used in Helm Chart deployment to generate cluster secret
    echo "REGISTRY_URL=${REGISTRY_URL}"
    echo "REGISTRY_NAMESPACE=${REGISTRY_NAMESPACE}"
    echo "GIT_BRANCH=${GIT_BRANCH}"
  } >> "${ARCHIVE_DIR}/build.properties"
  echo "File 'build.properties' created for passing env variables to subsequent pipeline jobs:"
  grep -v -i password "${ARCHIVE_DIR}/build.properties"
}

check_container_registry() {
  echo "=========================================================="
  echo "Checking registry current plan and quota"
  retry 3 3  ibmcloud cr login
  ibmcloud cr plan || true
  ibmcloud cr quota || true
  echo "If needed, discard older images using: ibmcloud cr image-rm"
  echo "Checking registry namespace: ${REGISTRY_NAMESPACE}"
  NS=$( ibmcloud cr namespaces | grep "${REGISTRY_NAMESPACE}" ||: )
  if [ -z "${NS}" ]; then
      echo "Registry namespace ${REGISTRY_NAMESPACE} not found, creating it."
      ibmcloud cr namespace-add "${REGISTRY_NAMESPACE}"
      echo "Registry namespace ${REGISTRY_NAMESPACE} created."
  else
      echo "Registry namespace ${REGISTRY_NAMESPACE} found."
  fi
  echo -e "Existing images in registry"
  ibmcloud cr images --restrict "${REGISTRY_NAMESPACE}/${IMAGE_NAME}"
}

prune_images() {
  KEEP=1
  echo -e "PURGING REGISTRY, only keeping last ${KEEP} image(s) based on image digests"
  COUNT=0
  LIST=$( ibmcloud cr images --restrict "${REGISTRY_NAMESPACE}/${IMAGE_NAME}" --no-trunc --format '{{ .Created }} {{ .Repository }}@{{ .Digest }}' | sort -r -u | awk '{print $2}' )
  while read -r IMAGE_URL ; do
    if [[ "$COUNT" -lt "$KEEP" ]]; then
      echo "Keeping image digest: ${IMAGE_URL}"
    else
      ibmcloud cr image-rm "${IMAGE_URL}"
    fi
    COUNT=$((COUNT+1))
  done <<< "$LIST"

  echo -e "Existing images in registry after clean up"
  ibmcloud cr images --restrict "${REGISTRY_NAMESPACE}/${IMAGE_NAME}"
}

build_image() {
  echo "=========================================================="
  echo -e "BUILDING CONTAINER IMAGE: ${IMAGE_NAME}:${IMAGE_TAG}"
  if [ -z "${DOCKER_ROOT}" ]; then DOCKER_ROOT=. ; fi
  if [ -z "${DOCKER_FILE}" ]; then DOCKER_FILE=Dockerfile ; fi
  BUILD_ARGS=()
  for buildArg in $BUILD_ARG_LIST; do
    BUILD_ARGS+=("--build-arg" "$buildArg")
  done

  ibmcloud cr build "${BUILD_ARGS[@]}" -f "${DOCKER_FILE}" --tag "${REGISTRY_URL}/${REGISTRY_NAMESPACE}/${IMAGE_NAME}:${IMAGE_TAG}" "${DOCKER_ROOT}"
  ibmcloud cr image-inspect "${REGISTRY_URL}/${REGISTRY_NAMESPACE}/${IMAGE_NAME}:${IMAGE_TAG}"
  ibmcloud cr images --restrict "${REGISTRY_NAMESPACE}/${IMAGE_NAME}"
}

print_env_vars

case "$RUN_TASK" in
  "artifact")
    artificat_for_next_stage
    ;;

  "image")
    check_container_registry
    prune_images
    build_image
    ;;

  *)
    echo "please specify RUN_TASK=artifact|image"
    ;;
esac
