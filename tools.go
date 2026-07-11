/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

//go:build tools

// Package tools pre-vendors the OpenTelemetry dependencies of the
// OpenCensus to OpenTelemetry migration (#1297). It is never compiled
// (the "tools" build tag is never set) and should be deleted once the
// migration lands and imports these packages for real.
package tools

import (
	_ "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	_ "github.com/prometheus/otlptranslator"
	_ "go.opentelemetry.io/otel/exporters/prometheus"
	_ "go.opentelemetry.io/otel/sdk/metric"
	_ "go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	_ "go.opentelemetry.io/otel/semconv/v1.34.0"
)
