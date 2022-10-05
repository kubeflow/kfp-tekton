#!/bin/bash

# Copyright 2020-2021 kubeflow.org
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

# *************************************************************************************************
#
#  Usage, running this script directly vs running it with Make.
#
#  i.e. compile report for all KFP samples with error statistics and no file listing:
#
#  - running the script directly:
#
#      VIRTUAL_ENV=temp/.venv ./test_kfp_samples.sh -a -e -s
#
#  - with Make you can run:
#
#      make report VENV=temp/.venv ALL_SAMPLES="TRUE" PRINT_ERRORS="TRUE" SKIP_FILES="TRUE"
#
# *************************************************************************************************

function help {
  bold=$(tput bold)
  normal=$(tput sgr0)
  color=$(tput setaf 6)
  echo
  echo "This scripts clones the ${bold}kubeflow/pipelines${normal} repository and attempts to compile each Python"
  echo "DSL script found in the compiler testdata directory, optionally including all samples."
  echo
  echo -e "${bold}USAGE:${normal}"
  echo -e "    $0 [${color}OPTIONS${normal}]"
  echo
  echo -e "${bold}OPTIONS:${normal}"
  grep -iE '\-\-[a-z-]+)\s+.*?# .*$$' "$0" | \
    awk -v color="${color}"\
        -v normal="${normal}" \
      'BEGIN {FS = ").*?# "}; {printf "%s%-35s%s%s\n", color, $1, normal, $2}'
  echo
}

# process command line parameters
while (( $# > 0 )); do
  case "$1" in
    -v|--kfp-version)          KFP_VERSION="$2";            shift 2 ;;  # KFP SDK version, default: 1.7.2
    -a|--include-all-samples)  ALL_SAMPLES="TRUE";          shift 1 ;;  # Compile all DSL scripts in KFP repo
    -s|--dont-list-files)      SKIP_FILES="TRUE";           shift 1 ;;  # Suppress compile status for each DSL file
    -e|--print-error-details)  PRINT_ERRORS="TRUE";         shift 1 ;;  # Print summary of compilation errors
    -h|--help)                 help;                        exit 0  ;;  # Show this help message
    -*)                        echo "Unknown option '$1'";  exit 1  ;;
    *)                         KFP_VERSION="$1";            break   ;;
  esac
done

# define global variables
KFP_GIT_VERSION=${KFP_VERSION:-1.8.4}
KFP_SDK_VERSION=${KFP_VERSION:-1.8.14}
KFP_REPO_URL="https://github.com/kubeflow/pipelines.git"
SCRIPT_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd)"
PROJECT_DIR="${TRAVIS_BUILD_DIR:-$(cd "${SCRIPT_DIR%/sdk/python/tests}"; pwd)}"
TEMP_DIR="${PROJECT_DIR}/temp"
VENV_DIR="${VIRTUAL_ENV:-${TEMP_DIR}/.venv}"
KFP_CLONE_DIR="${TEMP_DIR}/kubeflow/pipelines"
KFP_TESTDATA_DIR="${KFP_CLONE_DIR}/sdk/python/tests/compiler/testdata"
TEKTON_COMPILED_YAML_DIR="${TEMP_DIR}/tekton_compiler_output"
COMPILER_OUTPUTS_FILE="${TEMP_DIR}/test_kfp_samples_output.txt"
CONFIG_FILE="${PROJECT_DIR}/sdk/python/tests/config.yaml"
REPLACE_EXCEPTIONS="FALSE" # "TRUE" | "FALSE"

# print KFP SDK and GIT versions (might be different)
#echo "KFP_GIT_VERSION=${KFP_GIT_VERSION}"
#echo "KFP_SDK_VERSION=${KFP_SDK_VERSION}"

# show value of the _DIR variables, use set in POSIX mode in a sub-shell to supress functions definitions (https://stackoverflow.com/a/1305273/5601796)
#(set -o posix ; set) | grep "_DIR" | sort

# show which virtual environment being used
#env | grep VIRTUAL_ENV

mkdir -p "${TEMP_DIR}"
mkdir -p "${TEKTON_COMPILED_YAML_DIR}"

# don't override the testdata report when running report for all samples
if [[ "${ALL_SAMPLES}" == "TRUE" ]]; then
  COMPILE_REPORT_FILE="${TEMP_DIR}/test_kfp_samples_report_ALL.txt"
else
  COMPILE_REPORT_FILE="${PROJECT_DIR}/sdk/python/tests/test_kfp_samples_report.txt"
fi

# create a temporary copy of the previous compilation report
COMPILE_REPORT_FILE_OLD="${COMPILE_REPORT_FILE/%.txt/_before.txt}"
touch "${COMPILE_REPORT_FILE}"
cp "${COMPILE_REPORT_FILE}" "${COMPILE_REPORT_FILE_OLD}"

# clone kubeflow/pipeline repo to get the testdata DSL scripts
if [ ! -d "${KFP_CLONE_DIR}" ]; then
  git -c advice.detachedHead=false clone -b "${KFP_GIT_VERSION}" "${KFP_REPO_URL}" "${KFP_CLONE_DIR}" -q
else
  cd "${KFP_CLONE_DIR}"
  git fetch --all -q
  git -c advice.detachedHead=false checkout "${KFP_GIT_VERSION}" -f -q
  cd - &> /dev/null
fi
echo "KFP clone version: $(git --git-dir "${KFP_CLONE_DIR}"/.git tag --points-at HEAD)"

# check if we are running in a Python virtual environment, if not create one
if [ ! -d "${VENV_DIR}" ]; then
  echo "Creating Python virtual environment ..."
  python3 -m venv "${VENV_DIR}"
  source "${VENV_DIR}/bin/activate"
  pip install -q --upgrade pip
fi
source "${VENV_DIR}/bin/activate"

# install KFP with the desired KFP SDK version (unless already installed)
if ! (pip show "kfp" | grep Version | grep -q "${KFP_SDK_VERSION}"); then
  echo "Installing KFP SDK ${KFP_SDK_VERSION} ..."
  pip install -q kfp==${KFP_SDK_VERSION}
fi

# install KFP-Tekton compiler, unless already installed
if ! (pip show "kfp-tekton" | grep Location | grep -q "${PROJECT_DIR}"); then
  echo "Installing KFP-Tekton ..."
  pip install -q -e "${PROJECT_DIR}/sdk/python"
fi

# install pytest, unless already installed
if ! (pip show "pytest" | grep -q Version); then
  echo "Installing pytest ..."
  pip install -q pytest
fi

# install 3rd party dependencies required for certain pipeline samples
if [[ "${ALL_SAMPLES}" == "TRUE" ]]; then
  echo "Installing 3rd-party dependencies ..."
  pip show ai_pipeline_params   >/dev/null 2>&1 || pip install ai_pipeline_params
  pip show kfp-azure-databricks >/dev/null 2>&1 || pip install -e "${KFP_CLONE_DIR}/samples/contrib/azure-samples/kfp-azure-databricks"
  pip show kfp-arena            >/dev/null 2>&1 || pip install "http://kubeflow.oss-cn-beijing.aliyuncs.com/kfp-arena/kfp-arena-0.6.tar.gz"

  # reinstall KFP with the desired version to get all of its dependencies with their respective desired versions
  # pip uninstall -q -y kfp
  pip install -q kfp==${KFP_SDK_VERSION} --force-reinstall
fi

# confirm installed kfp version(s)
echo "KFP Python SDK version(s):"
pip list | grep kfp

echo  # just adding some separation for console output

# replace NotImplementedError with simple print out
if [[ "${REPLACE_EXCEPTIONS}" == "TRUE" ]]; then
  find "${PROJECT_DIR}"/sdk/python/kfp_tekton/compiler/*.py -type f -exec gsed -i 's/raise NotImplementedError(/print("NotImplementedError: "+/' {} \;
  find "${PROJECT_DIR}"/sdk/python/kfp_tekton/compiler/*.py -type f -exec gsed -i 's/raise ValueError(/print("ValueError: "+/' {} \;
fi

# delete the previous compiler output file
rm -f "${COMPILE_REPORT_FILE}"
rm -f "${COMPILER_OUTPUTS_FILE}"

# check which pipelines have special configurations
SPECIAL_PIPELINES=$(awk '/pipeline:/{print $NF}' "${CONFIG_FILE}")

function compile_dsl {
  IS_SPECIAL=$(grep -E "${1##*/}" <<< "${SPECIAL_PIPELINES}")
  if [ -z "${IS_SPECIAL}" ]; then
    dsl-compile-tekton --py "$1" --output "$2"
  else
    export PYTHONPATH="${PROJECT_DIR}/sdk/python/tests"
    python3 -m test_util "$1" "$2"
  fi
}

# find the pipeline DSL scripts in the KFP repository
# make newlines the only separator to support arrays and looping over files with spaces in their name
IFS=$'\n'
if [[ "${ALL_SAMPLES}" == "TRUE" ]]; then
  # find all the pipeline DSL scripts in the KFP repository
  CONTRIB_PIPELINES=$(find "${KFP_CLONE_DIR}" -name "*.py" -path "*/contrib/*" -not -path "*/.venv/*" -exec grep -i -l "dsl.Pipeline" {} + | sort)
  SAMPLE_PIPELINES=$(find "${KFP_CLONE_DIR}" -name "*.py" -not -path "*/contrib/*" -not -path "*/sdk/python/*" -not -path "*/.venv/*" -exec grep -i -l "dsl.Pipeline" {} + | sort)
  DSL_SCRIPTS=(
    "${KFP_TESTDATA_DIR}"/*.py
    ${SAMPLE_PIPELINES[@]}
    ${CONTRIB_PIPELINES[@]}
  )
else
  # only the pipelines in KFP compiler testdata
  DSL_SCRIPTS=("${KFP_TESTDATA_DIR}"/*.py)
fi

# run the KFP-Tekton compiler on the dsl.Pipeline scripts
i=1
for f in "${DSL_SCRIPTS[@]}"; do

  # display just the file name when compiling testdata scripts only, keep relative paths when compiling all KFP samples
  if [[ "${ALL_SAMPLES}" == "TRUE" ]]; then
    file_shortname="${f#${KFP_CLONE_DIR}/}"
  else
    file_shortname="${f##*/}"
  fi
  yaml_file="${TEKTON_COMPILED_YAML_DIR}/${f##*/}.yaml"

  echo -e "\nCompiling ${file_shortname}:" >> "${COMPILER_OUTPUTS_FILE}"

  # change directory to allow loading pipeline components from relative paths, set PYTHONPATH to local exec directory
  cd "${f%/*}"
  export PYTHONPATH="${f%/*}"

  # compile the DSL script
  if compile_dsl "${f}" "${yaml_file}" >> "${COMPILER_OUTPUTS_FILE}" 2>&1;
  then
    status="SUCCESS"
  else
    status="FAILURE"
  fi

  # print SUCCESS or FAILURE status to report file
  echo "${status}: ${file_shortname}" | tee -a "${COMPILE_REPORT_FILE}" >> "${COMPILER_OUTPUTS_FILE}"

  # print progress report to console
  if [[ "${SKIP_FILES}" == "TRUE"  ]]
  then
    echo -ne "\r\033[0KProgress: ${i}/${#DSL_SCRIPTS[@]}";
  else
    tail -1 "${COMPILE_REPORT_FILE}"
  fi

  # change back the working directory
  cd - &> /dev/null

  ((++i))
done

# add some space
[[ "${SKIP_FILES}" == "TRUE"  ]] && echo

# function to compile the success-failure-report
function compile_report() {
  FILE_GROUP="$1"
  FILE_FILTER="$2"

  SUCCESS=$( grep "${FILE_FILTER}" "${COMPILE_REPORT_FILE}" | grep -c "SUCCESS" )
  FAILURE=$( grep "${FILE_FILTER}" "${COMPILE_REPORT_FILE}" | grep -c "FAILURE" )
  TOTAL=$(   grep "${FILE_FILTER}" "${COMPILE_REPORT_FILE}" | grep -c "SUCCESS\|FAILURE" )
  (
    echo
    echo "Compilation status for ${FILE_GROUP}:"
    echo
    echo "  Success: ${SUCCESS}"
    echo "  Failure: ${FAILURE}"
    echo "  Total:   ${TOTAL}"
  )
}

# print success-failure-report summary in groups
if [[ "${ALL_SAMPLES}" == "TRUE" ]]; then
  compile_report "testdata DSL scripts" "/testdata/"
  compile_report "core samples" 'samples/core/\|samples/tutorials'
  compile_report "3rd-party contributed samples" 'contrib/samples/\|samples/contrib'
else
  compile_report "testdata DSL scripts" ".py"
fi

# print overall success-failure-report summary
SUCCESS=$( grep -c "SUCCESS" "${COMPILE_REPORT_FILE}" )
TOTAL=$(   grep -c "SUCCESS\|FAILURE" "${COMPILE_REPORT_FILE}")
SUCCESS_RATE=$(awk -v s="${SUCCESS}" -v t="${TOTAL}" 'BEGIN { printf("%.0f%%\n", 100.0/t*s) }')
echo
echo "Overall success rate: ${SUCCESS}/${TOTAL} = ${SUCCESS_RATE}"

# print error statistics
if [[ "${PRINT_ERRORS}" == "TRUE" ]]; then
  echo
  echo "Occurences of NotImplementedError:"
  grep "NotImplementedError: " "${COMPILER_OUTPUTS_FILE}" | sed 's/NotImplementedError: //' | sort | uniq -c | sort -n -r
  echo
  echo "Occurences of other Errors:"
  grep "Error: " "${COMPILER_OUTPUTS_FILE}" | grep -v "NotImplementedError" | sort | uniq -c | sort -n -r | grep "." || echo "   0"
fi

# display all output file locations
echo
echo "Compilation status report:   ${COMPILE_REPORT_FILE#${PROJECT_DIR}/}"
echo "Accumulated compiler logs:   ${COMPILER_OUTPUTS_FILE#${PROJECT_DIR}/}"
echo "Compiled Tekton YAML files:  ${TEKTON_COMPILED_YAML_DIR#${PROJECT_DIR}/}/"

# check for missing Python modules
if grep -q "ModuleNotFoundError:" "${COMPILER_OUTPUTS_FILE}"; then
  echo
  echo "NOTE: Please update this script to install required Python modules:"
  grep "ModuleNotFoundError:" "${COMPILER_OUTPUTS_FILE}" | sort | uniq | awk 'NF{ print " - " $NF }'
fi

# re-instate the NotImplementedErrors
if [[ "${REPLACE_EXCEPTIONS}" == "TRUE" ]]; then
  find "${PROJECT_DIR}"/sdk/python/kfp_tekton/compiler/*.py -type f -exec gsed -i 's/print("NotImplementedError: "+/raise NotImplementedError(/' {} \;
  find "${PROJECT_DIR}"/sdk/python/kfp_tekton/compiler/*.py -type f -exec gsed -i 's/print("ValueError: "+/raise ValueError(/' {} \;
fi

# for Travis/CI integration return exit code 1 if this report is different from the previous report
# sort the list of files since we cannot ensure same sort order on MacOS (local) and Linux (build machine)
if [[ ! "${ALL_SAMPLES}" == "TRUE" ]]; then
  if ! diff -q -a -w -B \
      <(sort "${COMPILE_REPORT_FILE}") \
      <(sort "${COMPILE_REPORT_FILE_OLD}") >/dev/null 2>&1
  then
    echo
    echo "This compilation report (left) differs from the previous report (right):"
    echo
    diff -y -W 80 --suppress-common-lines -d \
        <(sort -k2 "${COMPILE_REPORT_FILE}") \
        <(sort -k2 "${COMPILE_REPORT_FILE_OLD}")
    echo
    rm -f "${COMPILE_REPORT_FILE_OLD}"
    exit 1
  else
    echo
    echo "This compilation report did not change from the previous report."
    echo
    rm -f "${COMPILE_REPORT_FILE_OLD}"
    exit 0
  fi
fi
