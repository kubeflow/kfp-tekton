#!/bin/bash -e

# Copyright 2020 kubeflow.org
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


# The scripts clones the kubeflow/pipelines repository and attempts to compile
# each Python DSL script found in the compiler testdata directory.
#
# Usage:
#   ./test_kfp_samples.sh [KFP version, default to 0.2.2]

KFP_VERSION=${1:-0.2.2}
KFP_REPO_URL="https://github.com/kubeflow/pipelines.git"

SCRIPT_DIR="$(dirname "$0")"
PROJECT_DIR="$(cd "${SCRIPT_DIR%/sdk/python/tests}"; pwd)"
TEMP_DIR="${PROJECT_DIR}/temp"
VENV_DIR="${TEMP_DIR}/.venv"
KFP_CLONE_DIR="${TEMP_DIR}/kubeflow/pipelines"
KFP_TESTDATA_DIR="${KFP_CLONE_DIR}/sdk/python/tests/compiler/testdata"
TEKTON_COMPILED_YAML_DIR="${TEMP_DIR}/tekton_compiler_output"
COMPILE_REPORT_FILE="${PROJECT_DIR}/sdk/python/tests/test_kfp_samples_report.txt"
COMPILER_OUTPUTS_FILE="${TEMP_DIR}/test_kfp_samples_output.txt"

mkdir -p "${TEMP_DIR}"
mkdir -p "${TEKTON_COMPILED_YAML_DIR}"

KFP_REQ_VERSION=$(grep "kfp" "${PROJECT_DIR}/sdk/python/requirements.txt" | sed -n -E "s/^kfp[=<>]+([0-9.]+)$/\1/p")

if [ "$KFP_VERSION" != "$KFP_REQ_VERSION" ]; then
  echo "NOTE: the KFP version in the 'requirements.txt' file does not match"
  echo "  KFP_VERSION:      $KFP_VERSION"
  echo "  requirements.txt: $KFP_REQ_VERSION"
fi

if [ ! -d "${KFP_CLONE_DIR}" ]; then
  git clone -b "${KFP_VERSION}" "${KFP_REPO_URL}" "${KFP_CLONE_DIR}"
else
  cd "${KFP_CLONE_DIR}"
  git checkout "${KFP_VERSION}"
  cd - &> /dev/null
fi

if [ ! -d "${VENV_DIR}" ]; then
  echo "Creating Python virtual environment..."
  python3 -m venv "${VENV_DIR}"
  source "${VENV_DIR}/bin/activate"
  pip install --upgrade pip
fi

source "${VENV_DIR}/bin/activate"

if ! (pip show kfp | grep Location | grep -q "${KFP_CLONE_DIR}"); then
  pip install -e "${KFP_CLONE_DIR}/sdk/python"
fi

if ! (pip show kfp | grep Version | grep -q "${KFP_VERSION}"); then
  pip install -e "${KFP_CLONE_DIR}/sdk/python"
fi

if ! (pip show "kfp-tekton" | grep Location | grep -q "${PROJECT_DIR}"); then
  pip install -e "${PROJECT_DIR}/sdk/python"
fi

pip list | grep "kfp\|tekton" | grep -v "-server-api"

echo

rm -f "${COMPILER_OUTPUTS_FILE}"

for f in "${KFP_TESTDATA_DIR}"/*.py; do
  echo -e "\nCompiling ${f##*/}:" >> "${COMPILER_OUTPUTS_FILE}"
  if dsl-compile-tekton --py "${f}" --output "${TEKTON_COMPILED_YAML_DIR}/${f##*/}.yaml" >> "${COMPILER_OUTPUTS_FILE}" 2>&1;
  then
    echo "SUCCESS: ${f##*/}" | tee -a "${COMPILER_OUTPUTS_FILE}"
  else
    echo "FAILURE: ${f##*/}" | tee -a "${COMPILER_OUTPUTS_FILE}"
  fi
done | tee "${COMPILE_REPORT_FILE}"

SUCCESS=$(grep -c "SUCCESS" "${COMPILE_REPORT_FILE}")
FAILURE=$(grep -c "FAILURE" "${COMPILE_REPORT_FILE}")
TOTAL=$(grep -c . "${COMPILE_REPORT_FILE}")

(
  echo
  echo "Success: ${SUCCESS}"
  echo "Failure: ${FAILURE}"
  echo "Total:   ${TOTAL}"
) | tee -a "${COMPILE_REPORT_FILE}"

echo
echo "The compilation status report was stored in ${COMPILE_REPORT_FILE}"
echo "The accumulated console logs can be found in ${COMPILER_OUTPUTS_FILE}"
echo

deactivate
