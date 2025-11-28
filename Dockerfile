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

# "builder-base" can be overriden using dockerb buildx's --build-context flag,
# by users who want to use a different images for the builder. E.g. if you need to use an older OS
# to avoid dependencies on very recent glibc versions.
# E.g. of the param: --build-context builder-base=docker-image://golang:<something>@sha256:<something>
# Must override builder-base, not builder, since the latter is referred to later in the file and so must not be
# directly replaced. See here, and note that "stage" parameter mentioned there has been renamed to
# "build-context": https://github.com/docker/buildx/pull/904#issuecomment-1005871838
FROM golang:1.25.4-bookworm@sha256:e17419604b6d1f9bc245694425f0ec9b1b53685c80850900a376fb10cb0f70cb AS builder-base
FROM --platform=$BUILDPLATFORM builder-base AS builder

ARG TARGETARCH

ENV GOPATH=/gopath/
ENV PATH=$GOPATH/bin:$PATH

RUN apt-get update --fix-missing && apt-get --yes install libsystemd-dev
RUN go version

COPY . /gopath/src/k8s.io/node-problem-detector/
WORKDIR /gopath/src/k8s.io/node-problem-detector
RUN GOARCH=${TARGETARCH} make bin/node-problem-detector bin/health-checker bin/log-counter

FROM registry.k8s.io/build-image/debian-base:bookworm-v1.0.6@sha256:e100119ba6cf265b29957046f178e9374f9a9419133284c9b883e64c5b463d73 AS base

LABEL maintainer="Random Liu <lantaol@google.com>"

RUN clean-install util-linux bash libsystemd-dev

# Avoid symlink of /etc/localtime.
RUN test -h /etc/localtime && rm -f /etc/localtime && cp /usr/share/zoneinfo/UTC /etc/localtime || true

COPY --from=builder /gopath/src/k8s.io/node-problem-detector/bin/node-problem-detector /node-problem-detector

ARG LOGCOUNTER
COPY --from=builder /gopath/src/k8s.io/node-problem-detector/bin/health-checker /gopath/src/k8s.io/node-problem-detector/${LOGCOUNTER} /home/kubernetes/bin/

COPY --from=builder /gopath/src/k8s.io/node-problem-detector/config/ /config
ENTRYPOINT ["/node-problem-detector", "--config.system-log-monitor=/config/kernel-monitor.json,/config/readonly-monitor.json"]
