.PHONY: all container push clean node-problem-detector Dockerfile vet fmt

all: container

# See node-problem-detector.yaml for the version currently running-- bump this ahead before rebuilding!
# TAG is the tag of the container image.
TAG?=v0.3

# PROJ is the image project.
PROJ?=gcr.io/google_containers

# ENABLE_JOURNALD enables build journald support or not. Building journald support needs libsystemd-dev
# or libsystemd-journal-dev.
# TODO(random-liu): Build NPD inside container.
ENABLE_JOURNALD?=1

# TODO(random-liu): Support different architectures.
BASEIMAGE:=alpine:3.4

# Disable cgo by default to make the binary statically linked.
CGO_ENABLED:=0

# NOTE that enable journald will increase the image size.
ifeq ($(ENABLE_JOURNALD), 1)
	# Enable journald build tag.
	BUILD_TAGS:=-tags journald
	# Use fedora because it has newer systemd version (229) and support +LZ4. +LZ4 is needed
	# on some os distros such as GCI.
	BASEIMAGE:=fedora
	# Enable cgo because sdjournal needs cgo to compile. The binary will be dynamically
	# linked if CGO_ENABLED is enabled. This is fine because fedora already has necessary
	# dynamic library. We can not use `-extldflags "-static"` here, because go-systemd uses
	# dlopen, and dlopen will not work properly in a statically linked application.
	CGO_ENABLED:=1
endif

PKG_SOURCES := $(shell find pkg -name '*.go')

vet:
	go list ./... | grep -v "./vendor/*" | xargs go vet

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

node-problem-detector: $(PKG_SOURCES) node_problem_detector.go fmt vet
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux go build -a -ldflags '-w' $(BUILD_TAGS) -o node-problem-detector

Dockerfile: Dockerfile.in
	sed -e 's|@BASEIMAGE@|$(BASEIMAGE)|g' $< >$@

container: node-problem-detector Dockerfile
	docker build -t $(PROJ)/node-problem-detector:$(TAG) .

push: container
	gcloud docker push $(PROJ)/node-problem-detector:$(TAG)

clean:
	rm -f Dockerfile
	rm -f node-problem-detector

test:
	go test -timeout=1m -v -race ./pkg/... $(BUILD_TAGS)
