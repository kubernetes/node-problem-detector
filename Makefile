all: push

# See pod.yaml for the version currently running-- bump this ahead before rebuilding!
TAG = 0.1

# TODO(random-liu): Change the project to google_containers.
PROJ = google.com/noogler-kubernetes

node-problem-detector: node_problem_detector.go
	CGO_ENABLED=0 GOOS=linux godep go build -a -installsuffix cgo -ldflags '-w' -o node-problem-detector

container: node-problem-detector
	docker build -t gcr.io/$(PROJ)/node-problem-detector:$(TAG) .

push: container
	gcloud docker push gcr.io/$(PROJ)/node-problem-detector:$(TAG)

clean:
	rm -f node-problem-detector
