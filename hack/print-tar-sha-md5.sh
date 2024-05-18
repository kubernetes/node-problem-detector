#!/bin/bash

# Copyright 2024 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

VERSION="$1"

NPD_LINUX_AMD64=node-problem-detector-${VERSION}-linux_amd64.tar.gz
NPD_LINUX_ARM64=node-problem-detector-${VERSION}-linux_arm64.tar.gz
NPD_WINDOWS_AMD64=node-problem-detector-${VERSION}-windows_amd64.tar.gz

SHA_NPD_LINUX_AMD64=$(sha256sum ${NPD_LINUX_AMD64} | cut -d' ' -f1)
SHA_NPD_LINUX_ARM64=$(sha256sum ${NPD_LINUX_ARM64} | cut -d' ' -f1)
SHA_NPD_WINDOWS_AMD64=$(sha256sum ${NPD_WINDOWS_AMD64} | cut -d' ' -f1)

MD5_NPD_LINUX_AMD64=$(md5sum ${NPD_LINUX_AMD64} | cut -d' ' -f1)
MD5_NPD_LINUX_ARM64=$(md5sum ${NPD_LINUX_ARM64} | cut -d' ' -f1)
MD5_NPD_WINDOWS_AMD64=$(md5sum ${NPD_WINDOWS_AMD64} | cut -d' ' -f1)

echo
echo **${NPD_LINUX_AMD64}**:
echo **SHA**: ${SHA_NPD_LINUX_AMD64}
echo **MD5**: ${MD5_NPD_LINUX_AMD64}
echo
echo **${NPD_LINUX_ARM64}**:
echo **SHA**: ${SHA_NPD_LINUX_ARM64}
echo **MD5**: ${MD5_NPD_LINUX_ARM64}
echo
echo **${NPD_WINDOWS_AMD64}**:
echo **SHA**: ${SHA_NPD_WINDOWS_AMD64}
echo **MD5**: ${MD5_NPD_WINDOWS_AMD64}
