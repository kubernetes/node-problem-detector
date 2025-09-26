/*
Copyright 2019 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package otel

import (
	"testing"

	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

func TestGetResource(t *testing.T) {
	resource := GetResource()
	if resource == nil {
		t.Fatal("Expected resource to be created, got nil")
	}

	attrs := resource.Attributes()

	// Check service attributes
	var serviceName string
	var serviceVersion string

	for _, attr := range attrs {
		switch attr.Key {
		case semconv.ServiceNameKey:
			serviceName = attr.Value.AsString()
		case semconv.ServiceVersionKey:
			serviceVersion = attr.Value.AsString()
		}
	}
	if serviceName != "node-problem-detector" {
		t.Errorf("Expected service name 'node-problem-detector', got '%s'", serviceName)
	}

	if serviceVersion == "" {
		t.Error("Expected service version to be set")
	}
}
