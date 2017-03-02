#!/bin/bash

# Copyright 2017 The Kubernetes Authors.
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

# This script generate kernel log in given rate.
# Note that this only works for journald.
# Note that this script can't be used to test multi-line patterns, because
# the test kernel log lines may be splitted by real kernel log.

# PROBLEM is the path to the problem log.
PROBLEM=${PROBLEM:-""}
# LINES_PER_SECOND is the log generating rate.
LINES_PER_SECOND=${LINES_PER_SECOND:-1}
# LOG_PATH is the path to generate kernel logs.
LOG_PATH=${LOG_PATH:-"/dev/kmsg"}
# RUN_ONCE specifies whether to keep generating problem logs or only
# generate once.
RUN_ONCE=${RUN_ONCE:-"false"}

if [[ -z ${PROBLEM} ]]; then
  echo "$(basename $0): PROBLEM is empty"
  exit 1
fi

# now returns the current unix time in micro seconds
now() {
  echo $(($(date +'%s%N') / 1000))
}

prefix="kernel: "
logs=$(<${PROBLEM})
lines=$(printf "%s\n" "${logs}" | wc -l)
current_lines=0
start=$(now)
while true; do
  while read log; do
    echo "${prefix}${log}" >> ${LOG_PATH}
    ((current_lines++))
    if [[ ${current_lines} == ${LINES_PER_SECOND} ]]; then
      current_lines=0
      current=$(now)
      duration=$((1000000 - current + start))
      if [[ ${duration} -gt 0 ]]; then
        echo "generated ${LINES_PER_SECOND} lines of log, sleep for ${duration}"
        usleep ${duration}
      fi
      start=$(now)
    fi
  done <<< "$logs"
  if [[ "${RUN_ONCE}" == "true" ]]; then
    echo "stop the generator"
    exit 0
  fi
done
