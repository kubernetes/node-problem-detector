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
ROOT_PATH=$(git rev-parse --show-toplevel)
GCS_URL_PREFIX="https://storage.googleapis.com/"

function print-help() {
  echo "Usage: build.sh [args...]"
  echo "Available arguments:"
  echo "  pr [pull_number]: Build node-problem-detector for presubmit jobs and push to staging."
  echo "  ci:               Build node-problem-detector for CI jobs and push to staging."
  echo "  get-ci-env:       Download environment variable file from staging for CI job."
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

  set -x
  cat ${ROOT_PATH}/${env_file}
  set +x
}

function build-pr() {
  local -r PR_NUMBER="${1}"
  if [[ -z "${PR_NUMBER}" ]]; then
    echo "ERROR: pull_number is missing."
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
  export VERSION=$(get-version)
  export TAG="${VERSION}"
  make push
  write-env-file ${CI_ENV_FILENAME}
  gsutil mv ${ROOT_PATH}/${CI_ENV_FILENAME} ${UPLOAD_PATH}
}

function get-ci-env() {
  gsutil cp ${NPD_STAGING_PATH}/ci/${CI_ENV_FILENAME} ${ROOT_PATH}/${CI_ENV_FILENAME}
}


cd ${ROOT_PATH}

if [[ "$#" -ne 1 && "$#" -ne 2 ]]; then
  echo "ERROR: Illegal number of parameters."
  print-help
  exit 1
fi
COMMAND="${1}"

if [[ "${COMMAND}" == "pr" ]]; then
  if [[ "$#" -ne 2 ]]; then
    echo "ERROR: pull_number is missing."
    print-help
    exit 1
  fi
  build-pr "${2}"
elif [[ "${COMMAND}" == "ci" ]]; then
  build-ci
elif [[ "${COMMAND}" == "get-ci-env" ]]; then
  get-ci-env
elif [[ "${COMMAND}" == "install-lib" ]]; then
  install-lib
else
  echo "ERROR: Invalid command."
  print-help
  exit 1
fi
