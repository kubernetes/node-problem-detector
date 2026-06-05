#!/bin/bash

# Copyright 2026 The Kubernetes Authors.
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

# Side-loads the node-problem-detector image built by this PR/CI run onto a
# node e2e VM so the kubernetes "NodeProblemDetector" test exercises that image
# instead of pulling one from a registry.
#
# It is wired in as the instance startup-script by the ci-/pull-npd-e2e-node
# jobs, and reads the archive URL (uploaded by test/build.sh) from instance
# metadata. With --prepull-images=false and the pod's IfNotPresent policy, the
# imported image is used without any registry pull. No-op when the metadata is
# absent, so the job is safe to enable before build.sh starts publishing it.

set -o errexit
set -o nounset
set -o pipefail

readonly META="http://metadata.google.internal/computeMetadata/v1/instance/attributes"

url="$(curl -sf -H 'Metadata-Flavor: Google' "${META}/npd-image-url" || true)"
if [[ -z "${url}" ]]; then
  echo "npd-image-url instance metadata not set; nothing to side-load"
  exit 0
fi

echo "Downloading node-problem-detector image archive from ${url}"
curl -fsSL --retry 5 --retry-delay 3 -o /tmp/npd-image.tar "${url}"

# init.yaml restarts containerd near the end of node setup; wait for it before
# importing into the CRI (k8s.io) namespace that the kubelet uses.
for _ in $(seq 1 90); do
  if ctr -n k8s.io version >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo "Importing node-problem-detector image into containerd"
ctr -n k8s.io images import /tmp/npd-image.tar
ctr -n k8s.io images ls | grep node-problem-detector || true
