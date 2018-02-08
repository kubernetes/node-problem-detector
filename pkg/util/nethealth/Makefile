# Copyright 2016 The Kubernetes Authors All rights reserved.
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

# Makefile for the Docker image k8s.gcr.io/kube-nethealth-<ARCH>
# MAINTAINER: Girish kalele <abshah@google.com>
# If you update this image please bump the tag value before pushing.
#
# Usage:
#   [TAG=1.0] [REGISTRY=staging-k8s.gcr.io] make push

# Default registry, arch and tag. This can be overwritten by arguments to make
TAG?=1.0
REGISTRY?=staging-k8s.gcr.io
ARCH?=amd64

all: build

nethealth: nethealth.go
	CGO_ENABLED=0 go build -ldflags '-s' -v -o nethealth nethealth.go

build: nethealth
	docker build -t $(REGISTRY)/kube-nethealth-$(ARCH):$(TAG) .

push: build
ifeq ($(REGISTRY),staging-k8s.gcr.io)
	gcloud docker -- push $(REGISTRY)/kube-nethealth-$(ARCH):$(TAG)
else
	docker push $(REGISTRY)/kube-nethealth-$(ARCH):$(TAG)
endif

.PHONY: push
