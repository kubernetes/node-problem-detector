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

.PHONY: all all-push-tar all-build-tar all-container all-push node-problem-detector container push build-tar push-tar vet fmt test clean

# VERSION is the version of the binary.
VERSION:=$(shell git describe --tags --dirty)

ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

TAG ?= $(VERSION)
REGISTRY ?= gcr.io/google_containers
ARCH ?= $(shell go env GOARCH)
ALL_ARCH = amd64 arm arm64 ppc64le
QEMUVERSION=v2.9.1

GOARM=7
# List images with gcloud alpha container images list-tags gcr.io/google_containers/kube-cross
KUBE_CROSS_TAG=v1.8.3-1

IMGNAME = node-problem-detector
IMAGE = $(REGISTRY)/$(IMGNAME)
MULTI_ARCH_IMG = $(IMAGE)-$(ARCH)


# Set the (cross) compiler, BASEIMAGE to use for different architectures
ifeq ($(ARCH),amd64)
	CC=gcc
	BASEIMAGE:=fedora:26
endif
ifeq ($(ARCH),arm)
	QEMUARCH=arm
	CC=arm-linux-gnueabihf-gcc
	BASEIMAGE:=arm32v7/fedora:26
endif
ifeq ($(ARCH),arm64)
	QEMUARCH=aarch64
	CC=aarch64-linux-gnu-gcc
	BASEIMAGE:=arm64v8/fedora:26
endif
ifeq ($(ARCH),ppc64le)
	QEMUARCH=ppc64le
	CC=powerpc64le-linux-gnu-gcc
	BASEIMAGE:=ppc64le/fedora:26
endif
ifeq ($(ARCH),s390x)
	QEMUARCH=s390x
	CC=s390x-linux-gnu-gcc
	BASEIMAGE:=s390x/fedora:26
endif

# ENABLE_JOURNALD enables build journald support or not. Building journald support needs libsystemd-dev
# or libsystemd-journal-dev.
# TODO(random-liu): Build NPD inside container.
ENABLE_JOURNALD?=1

# Disable cgo by default to make the binary statically linked.
CGO_ENABLED:=0

# NOTE that enable journald will increase the image size.
ifeq ($(ENABLE_JOURNALD), 1)
	# Enable journald build tag.
	BUILD_TAGS:=-tags journald
	# Enable cgo because sdjournal needs cgo to compile. The binary will be dynamically
	# linked if CGO_ENABLED is enabled. This is fine because fedora already has necessary
	# dynamic library. We can not use `-extldflags "-static"` here, because go-systemd uses
	# dlopen, and dlopen will not work properly in a statically linked application.
	CGO_ENABLED:=1
endif

# UPLOAD_PATH is the cloud storage path to upload release tar.
UPLOAD_PATH?=gs://kubernetes-release

# Trim the trailing '/' in the path
UPLOAD_PATH:=$(shell echo $(UPLOAD_PATH) | sed '$$s/\/*$$//')

# PKG is the package name of node problem detector repo.
PKG:=k8s.io/node-problem-detector

# PKG_SOURCES are all the go source code.
PKG_SOURCES:=$(shell find pkg cmd -name '*.go')

TARBALL:=$(ROOT_DIR)/node-problem-detector-$(ARCH)-$(VERSION).tar.gz

TEMP_DIR := $(shell mktemp -d)

all: all-container

sub-container-%:
	$(MAKE) ARCH=$* container

sub-push-%:
	$(MAKE) ARCH=$* push

all-container: qemu-register $(addprefix sub-container-,$(ALL_ARCH))

all-push: $(addprefix sub-push-,$(ALL_ARCH))

node-problem-detector: $(PKG_SOURCES)
	docker run -e CC=$(CC) -e GOARM=$(GOARM) -e GOARCH=$(ARCH) \
		-v $(CURDIR):/go/src/k8s.io/node-problem-detector \
		-v $(TEMP_DIR):$(TEMP_DIR) \
		gcr.io/google_containers/kube-cross:$(KUBE_CROSS_TAG) /bin/bash -c "\
			apt-get -qq update && apt-get -qq install -y libsystemd-dev && \
			cd /go/src/k8s.io/node-problem-detector && \
			CGO_ENABLED=$(CGO_ENABLED) GOOS=linux go build -o $(TEMP_DIR)/bin/node-problem-detector \
			-ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
			$(BUILD_TAGS) cmd/node_problem_detector.go && \
			mkdir -p /go/src/k8s.io/node-problem-detector/bin && \
			cp $(TEMP_DIR)/bin/node-problem-detector /go/src/k8s.io/node-problem-detector/bin/node-problem-detector-$(ARCH)"

qemu-register:
	docker run --rm --privileged multiarch/qemu-user-static:register --reset

container: .container-$(ARCH)
.container-$(ARCH): node-problem-detector
	cp -r config $(TEMP_DIR)
	cp Dockerfile.in $(TEMP_DIR)/Dockerfile
	cd $(TEMP_DIR) && sed -i 's|@BASEIMAGE@|$(BASEIMAGE)|g' Dockerfile
	cd $(TEMP_DIR) && sed -i "s|ARCH|$(QEMUARCH)|g" Dockerfile

ifeq ($(ARCH),amd64)
	# When building "normally" for amd64, remove the whole line, it has no part in the amd64 image
	cd $(TEMP_DIR) && sed -i "/CROSS_BUILD_/d" Dockerfile
else
	curl -sSL https://github.com/multiarch/qemu-user-static/releases/download/$(QEMUVERSION)/x86_64_qemu-$(QEMUARCH)-static.tar.gz | tar -xz -C $(TEMP_DIR)
	cd $(TEMP_DIR) && sed -i "s/CROSS_BUILD_//g" Dockerfile
endif

	docker build -t $(MULTI_ARCH_IMG):$(TAG) $(TEMP_DIR)

ifeq ($(ARCH), amd64)
	# This is for to maintain the backward compatibility
	docker tag $(MULTI_ARCH_IMG):$(TAG) $(IMAGE):$(TAG)
endif

push: .push-$(ARCH)
.push-$(ARCH): .container-$(ARCH)
	gcloud docker -- push $(MULTI_ARCH_IMG):$(TAG)
ifeq ($(ARCH), amd64)
	gcloud docker -- push $(IMAGE):$(TAG)
endif

all-push-tar: $(addprefix sub-push-tar-,$(ALL_ARCH))
all-build-tar: $(addprefix sub-build-tar-,$(ALL_ARCH))

sub-push-tar-%:
	$(MAKE) ARCH=$* push-tar

sub-build-tar-%:
	$(MAKE) ARCH=$* build-tar

build-tar: node-problem-detector
	cp -r config $(TEMP_DIR)
	cd $(TEMP_DIR)
	tar -zcvf $(TARBALL) bin/ config/
	sha1sum $(TARBALL)
	md5sum $(TARBALL)

push-tar: build-tar
	gsutil cp $(TARBALL) $(UPLOAD_PATH)/node-problem-detector/

vet:
	go list ./... | grep -v "./vendor/*" | xargs go vet

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

test: vet fmt
	go test -timeout=1m -v -race ./pkg/... $(BUILD_TAGS)

clean:
	rm -f node-problem-detector-*.tar.gz
