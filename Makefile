.PHONY: all build-container build-tar build push-container push-tar push clean vet fmt version

all: build

VERSION := $(shell git describe --tags --dirty)

TAG ?= $(VERSION)

UPLOAD_PATH ?= gs://kubernetes-release
# Trim the trailing '/' in the path
UPLOAD_PATH := $(shell echo $(UPLOAD_PATH) | sed '$$s/\/*$$//')

PROJ ?= google_containers

PKG := k8s.io/node-problem-detector

PKG_SOURCES := $(shell find pkg cmd -name '*.go')

TARBALL := node-problem-detector-$(VERSION).tar.gz

IMAGE := gcr.io/$(PROJ)/node-problem-detector:$(TAG)

vet:
	go list ./... | grep -v "./vendor/*" | xargs go vet

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

version:
	@echo $(VERSION)

./bin/node-problem-detector: $(PKG_SOURCES)
	GOOS=linux go build -o bin/node-problem-detector \
	     -ldflags '-extldflags "-static" -X $(PKG)/pkg/version.version=$(VERSION)' \
	     cmd/node_problem_detector.go

test: vet fmt
	go test -timeout=1m -v -race ./pkg/...

build-container: ./bin/node-problem-detector
	docker build -t $(IMAGE) .

build-tar: ./bin/node-problem-detector
	tar -zcvf $(TARBALL) bin/ config/
	sha1sum $(TARBALL)
	md5sum $(TARBALL)

build: build-container build-tar

push-container: build-container
	gcloud docker push $(IMAGE)

push-tar: build-tar
	gsutil cp $(TARBALL) $(UPLOAD_PATH)/node-problem-detector/

push: push-container push-tar

clean:
	rm -f bin/node-problem-detector
	rm -f node-problem-detector-*.tar.gz
