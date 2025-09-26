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
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/pkg/version"
)

var (
	globalResource *resource.Resource
	resourceOnce   sync.Once
)

// GetResource returns the singleton OpenTelemetry resource with proper service identification
func GetResource() *resource.Resource {
	resourceOnce.Do(func() {
		globalResource = createResource()
	})
	return globalResource
}

// createResource creates the OpenTelemetry resource configuration
func createResource() *resource.Resource {
	// Create resource with minimal service info (no host/process details)
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			// Minimal service identification
			semconv.ServiceNameKey.String("node-problem-detector"),
			semconv.ServiceVersionKey.String(version.Version()),
			attribute.String("component", "node-problem-detector"),
		),
		resource.WithFromEnv(), // Allow environment overrides
	)
	if err != nil {
		klog.Errorf("Failed to create OpenTelemetry resource: %v", err)
		// Return a basic resource as fallback
		return resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("node-problem-detector"),
			semconv.ServiceVersionKey.String(version.Version()),
			attribute.String("component", "node-problem-detector"),
		)
	}

	return res
}
