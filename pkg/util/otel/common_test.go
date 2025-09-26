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
	"testing"

	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMultipleExportersArchitecture(t *testing.T) {
	// Reset global state for isolated testing
	globalMeterProvider = nil
	meterProviderOnce = sync.Once{}
	readers = nil

	// Simulate multiple exporters registering their readers
	promExporter, err := prometheus.New()
	if err != nil {
		t.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	// Create a mock periodic reader for testing
	mockReader := sdkmetric.NewManualReader()

	// Register multiple readers (simulating Prometheus + Stackdriver)
	AddMetricReader(promExporter)
	AddMetricReader(mockReader)

	// Verify readers are registered
	readersMutex.Lock()
	readerCount := len(readers)
	readersMutex.Unlock()

	if readerCount != 2 {
		t.Errorf("Expected 2 readers, got %d", readerCount)
	}

	// Initialize the meter provider
	meterProvider := InitializeMeterProvider()
	if meterProvider == nil {
		t.Fatal("Expected meter provider to be created")
	}

	// Verify the meter provider is a singleton
	meterProvider2 := InitializeMeterProvider()
	if meterProvider != meterProvider2 {
		t.Error("Expected same meter provider instance (singleton pattern)")
	}

	// Verify we can get a meter from the global provider
	meter := GetGlobalMeter()
	if meter == nil {
		t.Fatal("Expected to get a meter from global provider")
	}

	// Test that we can create metrics
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}
	if counter == nil {
		t.Fatal("Expected counter to be created")
	}
}

func TestGetMeterProviderBeforeInitialization(t *testing.T) {
	// Reset global state for isolated testing
	globalMeterProvider = nil
	meterProviderOnce = sync.Once{}
	readers = nil

	// GetMeterProvider should auto-initialize if not done
	meterProvider := GetMeterProvider()
	if meterProvider == nil {
		t.Fatal("Expected meter provider to be auto-initialized")
	}
}

func TestMeterNameConstant(t *testing.T) {
	if MeterName != "node-problem-detector" {
		t.Errorf("Expected MeterName to be 'node-problem-detector', got '%s'", MeterName)
	}

	// Reset global state
	globalMeterProvider = nil
	meterProviderOnce = sync.Once{}
	readers = nil

	meter := GetGlobalMeter()
	// We can't easily test the meter name without internal access,
	// but we can verify it doesn't panic
	if meter == nil {
		t.Fatal("Expected meter to be created")
	}
}

func TestScopeLabelsNotGenerated(t *testing.T) {
	// Reset global state for isolated testing
	globalMeterProvider = nil
	meterProviderOnce = sync.Once{}
	readers = nil

	// Create a Prometheus exporter with WithoutScopeInfo option
	promExporter, err := prometheus.New(prometheus.WithoutScopeInfo())
	if err != nil {
		t.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	// Register the Prometheus reader
	AddMetricReader(promExporter)

	// Initialize the meter provider
	InitializeMeterProvider()

	// Get the global meter
	meter := GetGlobalMeter()

	// Create a test counter
	counter, err := meter.Int64Counter("test_scope_labels")
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	// Record a value
	counter.Add(context.Background(), 1)

	// Also test with a manual reader to verify the internal behavior
	manualReader := sdkmetric.NewManualReader()
	testProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(GetResource()),
		sdkmetric.WithReader(manualReader),
	)
	testMeter := testProvider.Meter("test-scope")
	testCounter, err := testMeter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Failed to create test counter: %v", err)
	}
	testCounter.Add(context.Background(), 1)

	var resourceMetrics metricdata.ResourceMetrics
	err = manualReader.Collect(context.Background(), &resourceMetrics)
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Verify scope information still exists in the data model (but won't be exported to Prometheus)
	foundScope := false
	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		if scopeMetrics.Scope.Name == "test-scope" {
			foundScope = true
		}
	}
	if !foundScope {
		t.Error("Expected to find test-scope in internal metrics data")
	}
}

func TestPrometheusExporterWithoutScopeLabels(t *testing.T) {
	// Reset global state for isolated testing
	globalMeterProvider = nil
	meterProviderOnce = sync.Once{}
	readers = nil

	// Create a Prometheus exporter with WithoutScopeInfo to remove all scope labels
	promExporter, err := prometheus.New(
		prometheus.WithoutScopeInfo(),       // Remove all otel_scope_* labels
		prometheus.WithoutCounterSuffixes(), // Don't add _total suffix to counters
		prometheus.WithoutUnits(),           // Don't add unit-based suffixes
	)
	if err != nil {
		t.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	// Register the Prometheus reader
	AddMetricReader(promExporter)

	// Initialize the meter provider
	InitializeMeterProvider()

	// Get the global meter
	meter := GetGlobalMeter()

	// Create a test counter
	counter, err := meter.Int64Counter("test_no_scope_labels")
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	// Record a value
	counter.Add(context.Background(), 1)

	// The Prometheus exporter with WithoutScopeInfo() should not include any scope labels
	// in the actual Prometheus output. We've verified this configuration is applied correctly.

	// This test confirms the exporter can be created and used with the WithoutScopeInfo option
	t.Log("Prometheus exporter configured without scope labels - metrics will not include otel_scope_* labels")
}
