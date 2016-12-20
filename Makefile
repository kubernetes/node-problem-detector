.PHONY: all container hollow-container push hollow-push clean node-problem-detector

all: push

# See node-problem-detector.yaml for the version currently running-- bump this ahead before rebuilding!
TAG = v0.2

PROJ = google_containers

# TODO(shyamjvs) : Change this to google_containers once the hollow-node-problem-detector is tested fine.
TEST_PROJ = shyamjvs-k8s-km1

PKG_SOURCES := $(shell find pkg -name '*.go')

node-problem-detector: ./bin/node-problem-detector

./bin/node-problem-detector: $(PKG_SOURCES) node_problem_detector.go
	CGO_ENABLED=1 GOOS=linux godep go build -a -installsuffix cgo -ldflags '-w' -o ./bin/node-problem-detector

test:
	go test -timeout=1m -v -race ./pkg/...

container: node-problem-detector
	cp Dockerfile_template Dockerfile
	sed -i -e "s@{{config_directory}}@config@g" Dockerfile
	docker build -t gcr.io/$(PROJ)/node-problem-detector:$(TAG) .

hollow-container: node-problem-detector
	cp Dockerfile_template Dockerfile
	sed -i -e "s@{{config_directory}}@hollow_config@g" Dockerfile
	docker build -t gcr.io/$(TEST_PROJ)/node-problem-detector:$(TAG) .

push: container
	gcloud docker push gcr.io/$(PROJ)/node-problem-detector:$(TAG)

hollow-push: hollow-container
	gcloud docker push gcr.io/$(TEST_PROJ)/node-problem-detector:$(TAG)

clean:
	rm -f ./bin/node-problem-detector
	rm -f ./Dockerfile
