# Copyright 2016 The Kubernetes Authors All rights reserved.
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

# The debian-base:v1.0.0 image built from kubernetes repository is based on
# Debian Stretch. It includes systemd 232 with support for both +XZ and +LZ4
# compression. +LZ4 is needed on some os distros such as COS.
FROM k8s.gcr.io/debian-base-${TARGETARCH}:v2.0.0

ARG TARGETOS
ARG TARGETARCH
ARG LOGCOUNTER

MAINTAINER Random Liu <lantaol@google.com>

RUN clean-install util-linux libsystemd0 bash

# Avoid symlink of /etc/localtime.
RUN test -h /etc/localtime && rm -f /etc/localtime && cp /usr/share/zoneinfo/UTC /etc/localtime || true

COPY ./output/${TARGETOS}_${TARGETARCH}/bin/node-problem-detector /node-problem-detector
COPY ./output/${TARGETOS}_${TARGETARCH}/bin/health-checker ${LOGCOUNTER} /home/kubernetes/bin/

COPY config /config
ENTRYPOINT ["/node-problem-detector", "--config.system-log-monitor=/config/kernel-monitor.json"]
