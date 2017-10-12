#!/usr/bin/env bash

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

# This script is for installing node problem detector (NPD) on a running node
# in metric-only mode, as a setup for NPD e2e tests.

set -o errexit
set -o nounset
set -o pipefail

readonly BIN_DIR=/home/kubernetes/bin
readonly CONFIG_DIR=/home/kubernetes/node-problem-detector/config

function print-help() {
  echo "Usage: e2e-install.sh [flags] [command]"
  echo
  echo "Available flags:"
  echo "  -t [TARBALL]     Specify the path of the NPD tarball (generated by 'make build-tar')."
  echo
  echo "Available commands:"
  echo "  help     Print this help message"
  echo "  install  Installs NPD to the this machine"
  echo
  echo "Examples:"
  echo "  e2e-install.sh help"
  echo "  e2e-install.sh -t /tmp/npd.tar.gz install"
}

function install-npd() {
  if [[ -z "${TARBALL}" ]]; then
  	echo "ERROR: tarball flag is missing."
  	exit 1
  fi

  readonly workdir=$(mktemp -d)
  tar -xf "${TARBALL}" --directory "${workdir}"
  
  echo "Preparing NPD binary directory."
  mkdir -p "${BIN_DIR}"
  mount --bind "${BIN_DIR}" "${BIN_DIR}"
  # Below remount is to work around COS's noexec mount on /home.
  mount -o remount,exec "${BIN_DIR}"

  echo "Installing NPD binary."
  cp "${workdir}"/bin/node-problem-detector-amd64 "${BIN_DIR}"/node-problem-detector

  echo "Installing log-counter binary."
  cp "${workdir}"/bin/log-counter-amd64 "${BIN_DIR}"/log-counter

  echo "Installing NPD configurations."
  mkdir -p "${CONFIG_DIR}"
  cp -r "${workdir}"/config/* "${CONFIG_DIR}"

  echo "Installing NPD systemd service."
  cp "${workdir}"/config/systemd/node-problem-detector-metric-only.service /etc/systemd/system/node-problem-detector.service

  echo "Installing problem maker binary, used only for e2e testing."
  cp "${workdir}"/test/bin/problem-maker "${BIN_DIR}"

  rm -rf "${workdir}"

  # Start systemd service.
  echo "Starting NPD systemd service."
  systemctl daemon-reload
  systemctl stop node-problem-detector.service || true
  systemctl start node-problem-detector.service
}

function main() {
  case ${1:-} in
  help) print-help;;
  install) install-npd;;
  *) print-help;;
  esac
}

TARBALL=""

while getopts "t:" opt; do
  case ${opt} in
    t) TARBALL="${OPTARG}";;
  esac
done
shift "$((OPTIND-1))"


main "${@}"
