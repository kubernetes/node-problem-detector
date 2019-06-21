#!/bin/bash

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script build node problem detector for presubmit and CI jobs and push to
# staging, so that kubernetes E2E tests have access to the build.

set -o errexit
set -o nounset
set -o pipefail


NPD_STAGING_PATH=${NPD_STAGING_PATH:-"gs://node-problem-detector-staging"}
NPD_STAGING_REGISTRY=${NPD_STAGING_REGISTRY:-"gcr.io/node-problem-detector-staging"}
PR_ENV_FILENAME=${PR_ENV_FILENAME:-"pr.env"}
CI_ENV_FILENAME=${CI_ENV_FILENAME:-"ci.env"}
CI_CUSTOM_FLAGS_ENV_FILENAME=${CI_CUSTOM_FLAGS_ENV_FILENAME:-"ci-custom-flags.env"}
ROOT_PATH=$(git rev-parse --show-toplevel)
GCS_URL_PREFIX="https://storage.googleapis.com/"


function print-help() {
  echo "Usage: build.sh [flags] [command]"
  echo
  echo "Available flags:"
  echo "  -p [PR_NUMBER]      Specify the pull request number when building for a presubmit job."
  echo "  -f                  Use custom flags."
  echo
  echo "Available commands:"
  echo "  help           Print this help message"
  echo "  pr             Build node-problem-detector for presubmit jobs and push to staging. Flag -p is required."
  echo "  ci             Build node-problem-detector for CI jobs and push to staging."
  echo "  get-ci-env     Download environment variable file from staging for CI job."
  echo "  install-lib    Install the libraries needed."
  echo
  echo "Examples:"
  echo "  build.sh help"
  echo "  build.sh -p [PR_NUMBER] pr"
  echo "  build.sh -p [PR_NUMBER] -f pr"
  echo "  build.sh ci"
  echo "  build.sh get-ci-env"
  echo "  build.sh -f get-ci-env"
  echo "  build.sh install-lib"
}

function get-version() {
  if [ -d .git ]; then
    echo `git describe --tags --dirty`
  else
    echo "UNKNOWN"
  fi
}

function install-lib() {
  apt-get update
  apt-get install -y libsystemd-dev
}

function write-env-file() {
  local -r env_file="${1}"
  if [[ -z "${env_file}" ]]; then
    echo "ERROR: env_file is missing."
    exit 1
  fi

  cat > ${ROOT_PATH}/${env_file} <<EOF
export KUBE_ENABLE_NODE_PROBLEM_DETECTOR=standalone
export NODE_PROBLEM_DETECTOR_RELEASE_PATH=${UPLOAD_PATH/gs:\/\//${GCS_URL_PREFIX}}
export NODE_PROBLEM_DETECTOR_VERSION=${VERSION}
export NODE_PROBLEM_DETECTOR_TAR_HASH=$(sha1sum ${ROOT_PATH}/node-problem-detector-${VERSION}.tar.gz | cut -d ' ' -f1)
export EXTRA_ENVS=NODE_PROBLEM_DETECTOR_IMAGE=${REGISTRY}/node-problem-detector:${TAG}
EOF

  if [[ -n "${NODE_PROBLEM_DETECTOR_CUSTOM_FLAGS:-}" ]]; then
    cat >> ${ROOT_PATH}/${env_file} <<EOF
export NODE_PROBLEM_DETECTOR_CUSTOM_FLAGS="${NODE_PROBLEM_DETECTOR_CUSTOM_FLAGS}"
EOF
  fi

  echo "Written env file ${ROOT_PATH}/${env_file}:"
  cat ${ROOT_PATH}/${env_file}
}

function build-npd-custom-flags() {
  local -r kube_home="/home/kubernetes"

  local -r km_config="${kube_home}/node-problem-detector/config/kernel-monitor.json"
  local -r dm_config="${kube_home}/node-problem-detector/config/docker-monitor.json"
  local -r sm_config="${kube_home}/node-problem-detector/config/systemd-monitor.json"

  local -r custom_km_config="${kube_home}/node-problem-detector/config/kernel-monitor-counter.json"
  local -r custom_dm_config="${kube_home}/node-problem-detector/config/docker-monitor-counter.json"
  local -r custom_sm_config="${kube_home}/node-problem-detector/config/systemd-monitor-counter.json"

  flags="--v=2"
  flags+=" --logtostderr"
  flags+=" --system-log-monitors=${km_config},${dm_config},${sm_config}"
  flags+=" --custom-plugin-monitors=${custom_km_config},${custom_dm_config},${custom_sm_config}"
  flags+=" --port=20256"

  export NODE_PROBLEM_DETECTOR_CUSTOM_FLAGS=${flags}
}

function build-pr() {
  if [[ -z "${PR_NUMBER}" ]]; then
    echo "ERROR: PR_NUMBER is missing."
    print-help
    exit 1
  fi

  # Use the PR number and current time as the name, e.g., pr261-20190314.224907.862195792
  local -r PR="pr${PR_NUMBER}-$(date +%Y%m%d.%H%M%S.%N)"
  echo "Building for PR ${PR}..."
  install-lib

  export UPLOAD_PATH="${NPD_STAGING_PATH}/pr/${PR}"
  export REGISTRY="${NPD_STAGING_REGISTRY}/pr/${PR}"
  export VERSION=$(get-version)
  export TAG="${VERSION}"
  make push
  write-env-file ${PR_ENV_FILENAME}
}

function build-ci() {
  install-lib
  export UPLOAD_PATH="${NPD_STAGING_PATH}/ci"
  export REGISTRY="${NPD_STAGING_REGISTRY}/ci"
  export VERSION="$(get-version)-$(date +%Y%m%d.%H%M)"
  export TAG="${VERSION}"
  make push

  # Create the env file with and without custom flags at the same time.
  build-npd-custom-flags
  write-env-file ${CI_CUSTOM_FLAGS_ENV_FILENAME}
  gsutil mv ${ROOT_PATH}/${CI_CUSTOM_FLAGS_ENV_FILENAME} ${UPLOAD_PATH}

  export NODE_PROBLEM_DETECTOR_CUSTOM_FLAGS=""
  write-env-file ${CI_ENV_FILENAME}
  gsutil mv ${ROOT_PATH}/${CI_ENV_FILENAME} ${UPLOAD_PATH}
}

function get-ci-env() {
  if [[ "${USE_CUSTOM_FLAGS}" == "true" ]]; then
    gsutil cp ${NPD_STAGING_PATH}/ci/${CI_CUSTOM_FLAGS_ENV_FILENAME} ${ROOT_PATH}/${CI_CUSTOM_FLAGS_ENV_FILENAME}
    echo "Using env file ${ROOT_PATH}/${CI_CUSTOM_FLAGS_ENV_FILENAME}:"
    cat ${ROOT_PATH}/${CI_CUSTOM_FLAGS_ENV_FILENAME}
  else
    gsutil cp ${NPD_STAGING_PATH}/ci/${CI_ENV_FILENAME} ${ROOT_PATH}/${CI_ENV_FILENAME}
    echo "Using env file ${ROOT_PATH}/${CI_ENV_FILENAME}:"
    cat ${ROOT_PATH}/${CI_ENV_FILENAME}
  fi
}

main() {
  cd ${ROOT_PATH}

  if [[ "${USE_CUSTOM_FLAGS}" == "true" ]]; then
    build-npd-custom-flags
  fi

  # This is for the deprecated usage: build.sh pr [PR_NUMBER]
  if [[ -z "${PR_NUMBER}" ]]; then
    PR_NUMBER="${2:-}"
  fi

  case ${1:-} in
  help) print-help;;
  pr) build-pr;;
  ci) build-ci;;
  get-ci-env) get-ci-env;;
  install-lib) install-lib;;
  *) print-help;;
  esac
}


USE_CUSTOM_FLAGS="false"
PR_NUMBER=""

while getopts "fp:" opt; do
  case ${opt} in
    f) USE_CUSTOM_FLAGS="true";;
    p) PR_NUMBER="${OPTARG}";;
  esac
done
shift "$((OPTIND-1))"

main "$@"
