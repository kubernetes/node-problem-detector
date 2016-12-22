.PHONY: all container push clean node-problem-detector vet fmt

all: push

# See node-problem-detector.yaml for the version currently running-- bump this ahead before rebuilding!
TAG = v0.2

PROJ = google_containers

PKG_SOURCES := $(shell find pkg -name '*.go')

vet:
	go list ./... | grep -v "./vendor/*" | xargs go vet

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

node-problem-detector: $(PKG_SOURCES) node_problem_detector.go fmt vet
	GOOS=linux go build -ldflags '-w -extldflags "-static"' -o node-problem-detector

test:
	go test -timeout=1m -v -race ./pkg/...

container: node-problem-detector
	docker build -t gcr.io/$(PROJ)/node-problem-detector:$(TAG) .

push: container
	gcloud docker push gcr.io/$(PROJ)/node-problem-detector:$(TAG)

clean:
	rm -f node-problem-detector
