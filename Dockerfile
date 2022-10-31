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

ARG BASEIMAGE
#FROM k8s.gcr.io/debian-base-amd64:v2.0.0
FROM alpine:3.14

LABEL maintainer="Random Liu <lantaol@google.com>"

#RUN clean-install util-linux libsystemd0 bash systemd

# Avoid symlink of /etc/localtime.
RUN test -h /etc/localtime && rm -f /etc/localtime && cp /usr/share/zoneinfo/UTC /etc/localtime || true
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add --no-cache tcpdump lsof net-tools tzdata curl dumb-init libc6-compat bash iproute2 procps
RUN echo "hosts: files dns" > /etc/nsswitch.conf

COPY ./bin/node-problem-detector /node-problem-detector

ARG LOGCOUNTER
COPY ./bin/health-checker ./bin/log-counter /home/kubernetes/bin/

COPY config /config
ENTRYPOINT ["/node-problem-detector", "--config.system-log-monitor=/config/kernel-monitor.json"]
