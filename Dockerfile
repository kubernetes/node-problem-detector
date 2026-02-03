# "builder-base" can be overridden using docker buildx's --build-context flag,
# by users who want to use a different images for the builder. E.g. if you need to use an older OS
# to avoid dependencies on very recent glibc versions.
# E.g. of the param: --build-context builder-base=docker-image://golang:<something>@sha256:<something>
# Must override builder-base, not builder, since the latter is referred to later in the file and so must not be
# directly replaced. See here, and note that "stage" parameter mentioned there has been renamed to
# "build-context": https://github.com/docker/buildx/pull/904#issuecomment-1005871838
FROM golang:1.25.6-bookworm@sha256:2f768d462dbffbb0f0b3a5171009f162945b086f326e0b2a8fd5d29c3219ff14 AS builder-base
FROM --platform=$BUILDPLATFORM builder-base AS builder

ARG TARGETARCH

ENV GOPATH=/gopath/
ENV PATH=$GOPATH/bin:$PATH
RUN go version

RUN apt-get update --fix-missing && apt-get --yes install libsystemd-dev

COPY . /src/
WORKDIR /src
RUN GOARCH=${TARGETARCH} make bin/node-problem-detector bin/health-checker bin/log-counter

FROM registry.k8s.io/build-image/debian-base:bookworm-v1.0.7@sha256:368abceecc1308e0913a6fd5ab89a513ee4268becefc2a82dbe616462b29a46b AS base

RUN clean-install util-linux bash libsystemd-dev

# Avoid symlink of /etc/localtime.
RUN test -h /etc/localtime && rm -f /etc/localtime && cp /usr/share/zoneinfo/UTC /etc/localtime || true

COPY --from=builder /src/bin/node-problem-detector /node-problem-detector

ARG LOGCOUNTER
COPY --from=builder /src/bin/health-checker /src/${LOGCOUNTER} /home/kubernetes/bin/

COPY --from=builder /src/config/ /config
ENTRYPOINT ["/node-problem-detector", "--config.system-log-monitor=/config/kernel-monitor.json,/config/readonly-monitor.json"]
