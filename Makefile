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
        Dockerfile build-binaries docker-builder build-in-docker

all: build

# VERSION is the version of the binary.
VERSION?=$(shell if [ -d .git ]; then echo `git describe --tags --dirty`; else echo "UNKNOWN"; fi)

# TAG is the tag of the container image, default to binary version.
TAG?=$(VERSION)

# REGISTRY is the container registry to push into.
REGISTRY?=gcr.io/k8s-staging-npd

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
TARBALL?=node-problem-detector-$(VERSION).tar.gz

# IMAGE is the image name of the node problem detector container image.
IMAGE:=$(REGISTRY)/node-problem-detector:$(TAG)

# ENABLE_JOURNALD enables build journald support or not. Building journald
# support needs libsystemd-dev or libsystemd-journal-dev.
ENABLE_JOURNALD?=1

# TODO(random-liu): Support different architectures.
# The debian-base:v1.0.0 image built from kubernetes repository is based on
# Debian Stretch. It includes systemd 232 with support for both +XZ and +LZ4
# compression. +LZ4 is needed on some os distros such as COS.
BASEIMAGE:=k8s.gcr.io/debian-base-amd64:v1.0.0

# Disable cgo by default to make the binary statically linked.
CGO_ENABLED:=0

# Construct the "-tags" parameter used by "go build".
BUILD_TAGS?=

LINUX_BUILD_TAGS = $(BUILD_TAGS)
WINDOWS_BUILD_TAGS = $(BUILD_TAGS)

ifeq ($(ENABLE_JOURNALD), 1)
	# Enable journald build tag.
	LINUX_BUILD_TAGS := $(BUILD_TAGS) journald
	# Enable cgo because sdjournal needs cgo to compile. The binary will be
	# dynamically linked if CGO_ENABLED is enabled. This is fine because fedora
	# already has necessary dynamic library. We can not use `-extldflags "-static"`
	# here, because go-systemd uses dlopen, and dlopen will not work properly in a
	# statically linked application.
	CGO_ENABLED:=1
	LOGCOUNTER=./bin/log-counter
else
	# Hack: Don't copy over log-counter, use a wildcard path that shouldnt match
	# anything in COPY command.
	LOGCOUNTER=*dont-include-log-counter
endif

vet:
	GO111MODULE=on go list -mod vendor -tags "$(LINUX_BUILD_TAGS)" ./... | \
		grep -v "./vendor/*" | \
		GO111MODULE=on xargs go vet -mod vendor -tags "$(LINUX_BUILD_TAGS)"

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

version:
	@echo $(VERSION)

WINDOWS_AMD64_BINARIES = bin/windows_amd64/node-problem-detector.exe bin/windows_amd64/health-checker.exe
WINDOWS_AMD64_TEST_BINARIES = test/bin/windows_amd64/problem-maker.exe
LINUX_AMD64_BINARIES = bin/linux_amd64/node-problem-detector bin/linux_amd64/health-checker
LINUX_AMD64_TEST_BINARIES = test/bin/linux_amd64/problem-maker
ifeq ($(ENABLE_JOURNALD), 1)
	LINUX_AMD64_BINARIES += bin/linux_amd64/log-counter
endif

windows-binaries: $(WINDOWS_AMD64_BINARIES) $(WINDOWS_AMD64_TEST_BINARIES)

bin/windows_amd64/%.exe: $(PKG_SOURCES)
ifeq ($(ENABLE_JOURNALD), 1)
	echo "Journald on Windows is not supported, use make ENABLE_JOURNALD=0 [TARGET]"
endif
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) GO111MODULE=on go build \
		-mod vendor \
		-o $@ \
		-ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
		-tags "$(WINDOWS_BUILD_TAGS)" \
		./cmd/$(subst -,,$*)
	touch $@

./test/bin/windows_amd64/%.exe: $(PKG_SOURCES)
ifeq ($(ENABLE_JOURNALD), 1)
	echo "Journald on Windows is not supported, use make ENABLE_JOURNALD=0 [TARGET]"
endif
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) GO111MODULE=on go build \
		-mod vendor \
		-o $@ \
		-tags "$(WINDOWS_BUILD_TAGS)" \
		./test/e2e/$(subst -,,$*)

bin/linux_amd64/%: $(PKG_SOURCES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) GO111MODULE=on go build \
		-mod vendor \
		-o $@ \
		-ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
		-tags "$(LINUX_BUILD_TAGS)" \
		./cmd/$(subst -,,$*)
	touch $@

./test/bin/linux_amd64/%: $(PKG_SOURCES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) GO111MODULE=on go build \
		-mod vendor \
		-o $@ \
		-tags "$(LINUX_BUILD_TAGS)" \
		./test/e2e/$(subst -,,$*)

ifneq ($(ENABLE_JOURNALD), 1)
bin/linux_amd64/log-counter:
	echo "Warning: log-counter requires journald, skipping."
endif

# In the future these targets should be deprecated.
./bin/log-counter: $(PKG_SOURCES)
ifeq ($(ENABLE_JOURNALD), 1)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GO111MODULE=on go build \
		-mod vendor \
		-o bin/log-counter \
		-ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
		-tags "$(LINUX_BUILD_TAGS)" \
		cmd/logcounter/log_counter.go
else
	echo "Warning: log-counter requires journald, skipping."
endif

./bin/node-problem-detector: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GO111MODULE=on go build \
		-mod vendor \
		-o bin/node-problem-detector \
		-ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
		-tags "$(LINUX_BUILD_TAGS)" \
		./cmd/nodeproblemdetector

./test/bin/problem-maker: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GO111MODULE=on go build \
		-mod vendor \
		-o test/bin/problem-maker \
		-tags "$(LINUX_BUILD_TAGS)" \
		./test/e2e/problemmaker/problem_maker.go

./bin/health-checker: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GO111MODULE=on go build \
		-mod vendor \
		-o bin/health-checker \
		-ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
		-tags "$(LINUX_BUILD_TAGS)" \
		cmd/healthchecker/health_checker.go

test: vet fmt
	GO111MODULE=on go test -mod vendor -timeout=1m -v -race -short -tags "$(LINUX_BUILD_TAGS)" ./...

e2e-test: vet fmt build-tar
	GO111MODULE=on ginkgo -nodes=$(PARALLEL) -mod vendor -timeout=10m -v -tags "$(LINUX_BUILD_TAGS)" -stream \
	./test/e2e/metriconly/... -- \
	-project=$(PROJECT) -zone=$(ZONE) \
	-image=$(VM_IMAGE) -image-family=$(IMAGE_FAMILY) -image-project=$(IMAGE_PROJECT) \
	-ssh-user=$(SSH_USER) -ssh-key=$(SSH_KEY) \
	-npd-build-tar=`pwd`/$(TARBALL) \
	-boskos-project-type=$(BOSKOS_PROJECT_TYPE) -job-name=$(JOB_NAME) \
	-artifacts-dir=$(ARTIFACTS)

build-binaries: ./bin/node-problem-detector ./bin/log-counter ./bin/health-checker

build-container: build-binaries Dockerfile
	docker build -t $(IMAGE) --build-arg BASEIMAGE=$(BASEIMAGE) --build-arg LOGCOUNTER=$(LOGCOUNTER) .

build-tar: ./bin/node-problem-detector ./bin/log-counter ./bin/health-checker ./test/bin/problem-maker
	tar -zcvf $(TARBALL) bin/ config/ test/e2e-install.sh test/bin/problem-maker
	sha1sum $(TARBALL)
	md5sum $(TARBALL)

build: build-container build-tar

docker-builder:
	docker build -t npd-builder ./builder

build-in-docker: clean docker-builder
	docker run \
		-v `pwd`:/gopath/src/k8s.io/node-problem-detector/ npd-builder:latest bash \
		-c 'cd /gopath/src/k8s.io/node-problem-detector/ && make build-binaries'

push-container: build-container
	gcloud auth configure-docker
	docker push $(IMAGE)

push-tar: build-tar
	gsutil cp $(TARBALL) $(UPLOAD_PATH)/node-problem-detector/

push: push-container push-tar

clean:
	rm -f bin/health-checker
	rm -f bin/log-counter
	rm -f bin/node-problem-detector
	rm -f $(WINDOWS_AMD64_BINARIES) $(LINUX_AMD64_BINARIES)
	rm -f test/bin/problem-maker
	rm -f node-problem-detector-*.tar.gz
