.PHONY: all container push clean node-problem-detector vet fmt version

all: push

VERSION := $(shell git describe --tags --dirty)

TAG ?= $(VERSION)

PROJ = google_containers

PKG := k8s.io/node-problem-detector

PKG_SOURCES := $(shell find pkg cmd -name '*.go')

vet:
	go list ./... | grep -v "./vendor/*" | xargs go vet

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

version:
	@echo $(VERSION)

node-problem-detector: $(PKG_SOURCES) fmt vet
	GOOS=linux go build -o node-problem-detector \
	     -ldflags '-w -extldflags "-static" -X $(PKG)/pkg/version.version=$(VERSION)' \
	     cmd/node_problem_detector.go

test:
	go test -timeout=1m -v -race ./pkg/...

container: node-problem-detector
	docker build -t gcr.io/$(PROJ)/node-problem-detector:$(TAG) .

push: container
	gcloud docker push gcr.io/$(PROJ)/node-problem-detector:$(TAG)

clean:
	rm -f node-problem-detector
