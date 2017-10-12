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

.PHONY: all build-container build-tar build push-container push-tar push \
		clean vet fmt version \
        Dockerfile build-binaries docker-builder build-in-docker \
		build-all build-all-container \
        build-tar-all push-all push-all-container push-manifest push-tar-all qemu-register \
		release-staging

TEMP_DIR := $(shell mktemp -d)

all: build

# VERSION is the version of the binary.
VERSION?=$(shell if [ -d .git ]; then echo `git describe --tags --dirty`; else echo "UNKNOWN"; fi)

# TAG is the tag of the container image, default to binary version.
TAG?=$(VERSION)

ARCH ?= $(shell go env GOARCH)
ALL_ARCH = amd64 arm arm64 ppc64le s390x
QEMUVERSION=v4.0.0-4

# This option is for running docker manifest command
export DOCKER_CLI_EXPERIMENTAL := enabled

# REGISTRY is the container registry to push into.
REGISTRY?=staging-k8s.gcr.io

# STAGING_REGISTRY is the new registry for staging
STAGING_REGISTRY?=gcr.io/k8s-staging-npd

# UPLOAD_PATH is the cloud storage path to upload release tar.
UPLOAD_PATH?=gs://kubernetes-release
# Trim the trailing '/' in the path
UPLOAD_PATH:=$(shell echo $(UPLOAD_PATH) | sed '$$s/\/*$$//')

# PKG is the package name of node problem detector repo.
PKG:=k8s.io/node-problem-detector

# PKG_SOURCES are all the go source code.
PKG_SOURCES:=$(shell find pkg cmd -name '*.go')

# PARALLEL specifies the number of parallel test nodes to run for e2e tests.
PARALLEL?=3

# TARBALL is the name of release tar. Include binary version by default.
TARBALL:=node-problem-detector-$(VERSION).tar.gz
TARBALL-ALL:=node-problem-detector-$(VERSION)-all.tar.gz

# IMAGE is the image name of the node problem detector container image.
IMAGE:=$(REGISTRY)/node-problem-detector:$(TAG)
IMAGE_WITH_ARCH:=$(IMAGE)-$(ARCH)

# ENABLE_JOURNALD enables build journald support or not. Building journald
# support needs libsystemd-dev or libsystemd-journal-dev.
ENABLE_JOURNALD?=1

# The debian-base:v1.0.0 image built from kubernetes repository is based on
# Debian Stretch. It includes systemd 232 with support for both +XZ and +LZ4
# compression. +LZ4 is needed on some os distros such as COS.
# Set the (cross) compiler, BASEIMAGE to use for different architectures
ifeq ($(ARCH),amd64)
	CC=gcc
	BASEIMAGE:=k8s.gcr.io/debian-base-amd64:v1.0.0
endif
ifeq ($(ARCH),arm)
	QEMUARCH=arm
	CC=arm-linux-gnueabihf-gcc
	BASEIMAGE:=k8s.gcr.io/debian-base-arm:v1.0.0
endif
ifeq ($(ARCH),arm64)
	QEMUARCH=aarch64
	CC=aarch64-linux-gnu-gcc
	BASEIMAGE:=k8s.gcr.io/debian-base-arm64:v1.0.0
endif
ifeq ($(ARCH),ppc64le)
	QEMUARCH=ppc64le
	CC=powerpc64le-linux-gnu-gcc
	BASEIMAGE:=k8s.gcr.io/debian-base-ppc64le:v1.0.0
endif
ifeq ($(ARCH),s390x)
	QEMUARCH=s390x
	CC=s390x-linux-gnu-gcc
	BASEIMAGE:=k8s.gcr.io/debian-base-s390x:v1.0.0
endif

# Disable cgo by default to make the binary statically linked.
CGO_ENABLED:=0

# Construct the "-tags" parameter used by "go build".
BUILD_TAGS?=
ifeq ($(ENABLE_JOURNALD), 1)
	# Enable journald build tag.
	BUILD_TAGS:=$(BUILD_TAGS) journald
	# Enable cgo because sdjournal needs cgo to compile. The binary will be
	# dynamically linked if CGO_ENABLED is enabled. This is fine because fedora
	# already has necessary dynamic library. We can not use `-extldflags "-static"`
	# here, because go-systemd uses dlopen, and dlopen will not work properly in a
	# statically linked application.
	CGO_ENABLED:=1
endif

vet:
	GO111MODULE=on go list -mod vendor -tags "$(BUILD_TAGS)" ./... | \
		grep -v "./vendor/*" | \
		GO111MODULE=on xargs go vet -mod vendor -tags "$(BUILD_TAGS)"

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

version:
	@echo $(VERSION)

./bin/log-counter: $(PKG_SOURCES)
ifeq ($(ENABLE_JOURNALD), 1)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=$(ARCH) GO111MODULE=on go build \
		-mod vendor \
		-o bin/log-counter-$(ARCH) \
		-ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
		-tags "$(BUILD_TAGS)" \
		cmd/logcounter/log_counter.go
else
	echo "Warning: log-counter requires journald, skipping."
endif

./bin/node-problem-detector: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=$(ARCH) GO111MODULE=on go build \
		-mod vendor \
		-o bin/node-problem-detector-$(ARCH) \
		-ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
		-tags "$(BUILD_TAGS)" \
		./cmd/nodeproblemdetector

./test/bin/problem-maker: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=$(ARCH) GO111MODULE=on go build \
		-mod vendor \
		-o test/bin/problem-maker \
		-tags "$(BUILD_TAGS)" \
		./test/e2e/problemmaker/problem_maker.go

qemu-register:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

Dockerfile: Dockerfile.in
	sed -e 's|@BASEIMAGE@|$(BASEIMAGE)|g' $< >$@
ifneq ($(ENABLE_JOURNALD), 1)
	sed -i '/Below command depends on ENABLE_JOURNAL=1/,+2d' $@
	echo "Warning: log-counter requires journald, skipping."
endif
	sed -i "s|-ARCH|-$(QEMUARCH)|g" $@
	sed -i "s|-BINARCH|-$(ARCH)|g" $@
ifeq ($(ARCH),amd64)
	# When building "normally" for amd64, remove the whole line, it has no part in the amd64 image
	sed -i "/CROSS_BUILD_/d" Dockerfile
else
	curl -sSL https://github.com/multiarch/qemu-user-static/releases/download/$(QEMUVERSION)/x86_64_qemu-$(QEMUARCH)-static.tar.gz | tar -xz -C .
	sed -i "s/CROSS_BUILD_//g" Dockerfile
endif

test: vet fmt
	GO111MODULE=on go test -mod vendor -timeout=1m -v -race -short -tags "$(BUILD_TAGS)" ./...

e2e-test: vet fmt build-tar
	GO111MODULE=on ginkgo -nodes=$(PARALLEL) -mod vendor -timeout=10m -v -tags "$(BUILD_TAGS)" -stream \
	./test/e2e/metriconly/... -- \
	-project=$(PROJECT) -zone=$(ZONE) \
	-image=$(VM_IMAGE) -image-family=$(IMAGE_FAMILY) -image-project=$(IMAGE_PROJECT) \
	-ssh-user=$(SSH_USER) -ssh-key=$(SSH_KEY) \
	-npd-build-tar=`pwd`/$(TARBALL) \
	-boskos-project-type=$(BOSKOS_PROJECT_TYPE) -job-name=$(JOB_NAME) \
	-artifacts-dir=$(ARTIFACTS)

build-binaries: ./bin/node-problem-detector ./bin/log-counter

build-all-container: $(addprefix build-container-,$(ALL_ARCH))

build-container-%: qemu-register
	$(MAKE) ARCH=$* build-container

build-container: clean build-in-docker Dockerfile
	docker build -t $(IMAGE_WITH_ARCH) .

build-tar: ./bin/node-problem-detector ./bin/log-counter ./test/bin/problem-maker
	tar -zcvf $(TARBALL) bin/ config/ test/e2e-install.sh test/bin/problem-maker
	sha1sum $(TARBALL)
	md5sum $(TARBALL)

build-tar-all: clean
	@for arch in $(ALL_ARCH); do $(MAKE) ARCH=$${arch} build-in-docker; done
	@for arch in $(ALL_ARCH); do cp bin/node-problem-detector-$(ARCH) bin/node-problem-detector; done
	@for arch in $(ALL_ARCH); do cp bin/log-counter-$(ARCH) bin/log-counter; done
	tar -zcvf $(TARBALL-ALL) bin/ config/ test/e2e-install.sh
	sha1sum $(TARBALL-ALL)
	md5sum $(TARBALL-ALL)

build-all: build-all-container build-tar-all

build: build-container build-tar

docker-builder:
	docker build -t npd-builder ./builder

build-in-docker: docker-builder
	docker run \
		-e CC=$(CC) -e GOARM=$(GOARM) -e GOARCH=$(ARCH) -e GOCACHE=/go/.cache/go-build \
		-u $(shell id -u ${USER}):$(shell id -g ${USER}) \
		-v `pwd`:/gopath/src/k8s.io/node-problem-detector/ npd-builder:latest bash \
		-c 'cd /gopath/src/k8s.io/node-problem-detector/ && make build-binaries'

push-all-container: build-all-container $(addprefix push-container-,$(ALL_ARCH))

push-container-%:
	$(MAKE) ARCH=$* push-container

push-container: build-container
	gcloud auth configure-docker
	docker push $(IMAGE_WITH_ARCH)

push-manifest:
	gcloud auth configure-docker
	docker manifest create --amend $(IMAGE) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(IMAGE)\-&~g")
	@for arch in $(ALL_ARCH); do docker manifest annotate --arch $${arch} ${IMAGE} ${IMAGE}-$${arch}; done
	docker manifest push --purge ${IMAGE}

push-tar: build-tar
	gsutil cp $(TARBALL) $(UPLOAD_PATH)/node-problem-detector/

push-tar-all: build-tar-all
	gsutil cp $(TARBALL-ALL) $(UPLOAD_PATH)/node-problem-detector/

push-all: push-all-container push-manifest push-tar-all push-tar

push: push-container push-tar

release-staging:
	REGISTRY=$(STAGING_REGISTRY) $(MAKE) push-all

clean:
	rm -f bin/log-counter*
	rm -f bin/node-problem-detector*
	rm -f test/bin/problem-maker*
	rm -f node-problem-detector-*.tar.gz
