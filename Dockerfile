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

FROM golang:1.21.1-bookworm@sha256:d3114db136e7f2d9b08c3fcd374381ba3f8d85c17b80c48dbc53cd938cfb2b64 as builder
LABEL maintainer="Andy Xie <andy.xning@gmail.com>"

ARG TARGETARCH

ENV GOPATH /gopath/
ENV PATH $GOPATH/bin:$PATH

RUN apt-get update --fix-missing && apt-get --yes install libsystemd-dev gcc-aarch64-linux-gnu
RUN go version

COPY . /gopath/src/k8s.io/node-problem-detector/
WORKDIR /gopath/src/k8s.io/node-problem-detector
RUN GOARCH=${TARGETARCH} make bin/node-problem-detector bin/health-checker bin/log-counter

ARG BASEIMAGE
FROM --platform=${TARGETPLATFORM} ${BASEIMAGE}

LABEL maintainer="Random Liu <lantaol@google.com>"

RUN clean-install util-linux bash libsystemd-dev

# Avoid symlink of /etc/localtime.
RUN test -h /etc/localtime && rm -f /etc/localtime && cp /usr/share/zoneinfo/UTC /etc/localtime || true

COPY --from=builder /gopath/src/k8s.io/node-problem-detector/bin/node-problem-detector /node-problem-detector

ARG LOGCOUNTER
COPY --from=builder /gopath/src/k8s.io/node-problem-detector/bin/health-checker /gopath/src/k8s.io/node-problem-detector/${LOGCOUNTER} /home/kubernetes/bin/

COPY --from=builder /gopath/src/k8s.io/node-problem-detector/config/ /config
ENTRYPOINT ["/node-problem-detector", "--config.system-log-monitor=/config/kernel-monitor.json"]
