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

# Build the node-problem-detector image.

.PHONY: all build-container build-tar build push-container push-tar push clean vet fmt version \
        Dockerfile build-binaries docker-builder build-in-docker

all: build

# VERSION is the version of the binary.
VERSION?=$(shell if [ -d .git ]; then echo `git describe --tags --dirty`; else echo "UNKNOWN"; fi)

# TAG is the tag of the container image, default to binary version.
TAG?=$(VERSION)

# REGISTRY is the container registry to push into.
REGISTRY?=staging-k8s.gcr.io

# UPLOAD_PATH is the cloud storage path to upload release tar.
UPLOAD_PATH?=gs://kubernetes-release
# Trim the trailing '/' in the path
UPLOAD_PATH:=$(shell echo $(UPLOAD_PATH) | sed '$$s/\/*$$//')

# PKG is the package name of node problem detector repo.
PKG:=k8s.io/node-problem-detector

# PKG_SOURCES are all the go source code.
PKG_SOURCES:=$(shell find pkg cmd -name '*.go')

# TARBALL is the name of release tar. Include binary version by default.
TARBALL:=node-problem-detector-$(VERSION).tar.gz

# IMAGE is the image name of the node problem detector container image.
IMAGE:=$(REGISTRY)/node-problem-detector:$(TAG)

# ENABLE_JOURNALD enables build journald support or not. Building journald support needs libsystemd-dev
# or libsystemd-journal-dev.
ENABLE_JOURNALD?=1

# TODO(random-liu): Support different architectures.
# The debian-base:0.3 image built from kubernetes repository is based on Debian Stretch.
# It includes systemd 232 with support for both +XZ and +LZ4 compression.
# +LZ4 is needed on some os distros such as COS.
BASEIMAGE:=k8s.gcr.io/debian-base-amd64:0.3

# Disable cgo by default to make the binary statically linked.
CGO_ENABLED:=0

ifeq ($(ENABLE_JOURNALD), 1)
	# Enable journald build tag.
	BUILD_TAGS:=-tags journald
	# Enable cgo because sdjournal needs cgo to compile. The binary will be dynamically
	# linked if CGO_ENABLED is enabled. This is fine because fedora already has necessary
	# dynamic library. We can not use `-extldflags "-static"` here, because go-systemd uses
	# dlopen, and dlopen will not work properly in a statically linked application.
	CGO_ENABLED:=1
endif

vet:
	go list ./... | grep -v "./vendor/*" | xargs go vet $(BUILD_TAGS)

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

version:
	@echo $(VERSION)

./bin/log-counter: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux go build -o bin/log-counter \
	     -ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
	     $(BUILD_TAGS) cmd/logcounter/log_counter.go

./bin/node-problem-detector: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux go build -o bin/node-problem-detector \
	     -ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
	     $(BUILD_TAGS) cmd/node_problem_detector.go

Dockerfile: Dockerfile.in
	sed -e 's|@BASEIMAGE@|$(BASEIMAGE)|g' $< >$@

test: vet fmt
	go test -timeout=1m -v -race ./cmd/options ./pkg/... $(BUILD_TAGS)

build-binaries: ./bin/node-problem-detector ./bin/log-counter

build-container: build-binaries Dockerfile
	docker build -t $(IMAGE) .

build-tar: ./bin/node-problem-detector ./bin/log-counter
	tar -zcvf $(TARBALL) bin/ config/
	sha1sum $(TARBALL)
	md5sum $(TARBALL)

build: build-container build-tar

docker-builder:
	docker build -t npd-builder ./builder

build-in-docker: clean docker-builder
	docker run -v `pwd`:/gopath/src/k8s.io/node-problem-detector/ npd-builder:latest bash -c 'cd /gopath/src/k8s.io/node-problem-detector/ && make build-binaries'

push-container: build-container
	gcloud docker -- push $(IMAGE)

push-tar: build-tar
	gsutil cp $(TARBALL) $(UPLOAD_PATH)/node-problem-detector/

push: push-container push-tar

clean:
	rm -f bin/log-counter
	rm -f bin/node-problem-detector
	rm -f node-problem-detector-*.tar.gz
