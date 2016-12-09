.PHONY: all container push clean node-problem-detector Dockerfile

all: container

# See node-problem-detector.yaml for the version currently running-- bump this ahead before rebuilding!
# TAG is the tag of the container image.
TAG?=v0.3

# PROJ is the image project.
PROJ?=google_containers

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
	# dynamic library. If we need to make this statically linked in the future, add -ldflags
	# '-extldflags "-static"'
	CGO_ENABLED:=1
endif

PKG_SOURCES := $(shell find pkg -name '*.go')

node-problem-detector: ./bin/node-problem-detector

./bin/node-problem-detector: $(PKG_SOURCES) node_problem_detector.go
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux go build -o ./bin/node-problem-detector -a -ldflags '-w' $(BUILD_TAGS)

Dockerfile: Dockerfile.in
	sed -e 's|@BASEIMAGE@|$(BASEIMAGE)|g' $< >$@

container: node-problem-detector Dockerfile
	docker build -t gcr.io/$(PROJ)/node-problem-detector:$(TAG) .

push: container
	gcloud docker push gcr.io/$(PROJ)/node-problem-detector:$(TAG)

clean:
	rm -f Dockerfile
	rm -f ./bin/node-problem-detector

test:
	go test -timeout=1m -v -race ./pkg/... $(BUILD_TAGS)
