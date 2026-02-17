/*
Copyright 2025 The Kubernetes Authors All rights reserved.

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

package metrics

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

func TestFloat64GaugeSetValueSemantics(t *testing.T) {
	// Reset global state for isolated testing
	otelutil.ResetForTesting()

	// Set up SDK with ManualReader for testing our metrics
	reader := sdkmetric.NewManualReader()

	// Register reader with our global meter provider for our metrics to use
	otelutil.AddMetricReader(reader)
	otelutil.InitializeMeterProvider()

	// Create a gauge using our actual NewFloat64Metric function
	gauge, err := NewFloat64Metric(
		"test_float_gauge",
		"test_float_gauge",
		"Test float64 gauge metric",
		"percent",
		LastValue,
		[]string{"component", "state"},
	)
	if err != nil {
		t.Fatalf("Failed to create gauge metric: %v", err)
	}

	ctx := context.Background()
	labels := map[string]string{
		"component": "cpu",
		"state":     "usage",
	}

	// Set initial value to 0.0 (initialization)
	if err := gauge.Record(labels, 0.0); err != nil {
		t.Fatalf("Failed to record initial value: %v", err)
	}

	// Set value to 42.5 (current reading)
	if err := gauge.Record(labels, 42.5); err != nil {
		t.Fatalf("Failed to record updated value: %v", err)
	}

	// Collect metrics and verify gauge shows the last value (42.5)
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Find our metric in the collected data
	foundMetric := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test_float_gauge" {
				foundMetric = true

				expected := metricdata.Metrics{
					Name:        "test_float_gauge",
					Description: "Test float64 gauge metric",
					Unit:        "percent",
					Data: metricdata.Gauge[float64]{
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(
									attribute.String("component", "cpu"),
									attribute.String("state", "usage"),
								),
								Value: 42.5,
							},
						},
					},
				}

				metricdatatest.AssertEqual(t, expected, m, metricdatatest.IgnoreTimestamp())
			}
		}
	}

	if !foundMetric {
		t.Fatal("test_float_gauge metric not found in collected metrics")
	}

	// Set value to 15.3 (new reading)
	if err := gauge.Record(labels, 15.3); err != nil {
		t.Fatalf("Failed to record new value: %v", err)
	}

	// Collect again and verify gauge now shows 15.3
	rm = metricdata.ResourceMetrics{}
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	foundMetric = false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test_float_gauge" {
				foundMetric = true

				expected := metricdata.Metrics{
					Name:        "test_float_gauge",
					Description: "Test float64 gauge metric",
					Unit:        "percent",
					Data: metricdata.Gauge[float64]{
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(
									attribute.String("component", "cpu"),
									attribute.String("state", "usage"),
								),
								Value: 15.3,
							},
						},
					},
				}

				metricdatatest.AssertEqual(t, expected, m, metricdatatest.IgnoreTimestamp())
			}
		}
	}

	if !foundMetric {
		t.Fatal("test_float_gauge metric not found in second collection")
	}
}

func TestFloat64CounterAddSemantics(t *testing.T) {
	// Reset global state for isolated testing
	otelutil.ResetForTesting()

	// Set up SDK with ManualReader for testing our metrics
	reader := sdkmetric.NewManualReader()

	// Register reader with our global meter provider for our metrics to use
	otelutil.AddMetricReader(reader)
	otelutil.InitializeMeterProvider()

	// Create a counter using our actual NewFloat64Metric function
	counter, err := NewFloat64Metric(
		"test_float_counter",
		"test_float_counter",
		"Test float64 counter metric",
		"bytes",
		Sum,
		[]string{"operation"},
	)
	if err != nil {
		t.Fatalf("Failed to create counter metric: %v", err)
	}

	ctx := context.Background()
	labels := map[string]string{
		"operation": "read",
	}

	// Add to counter multiple times with fractional values using our Record method
	if err := counter.Record(labels, 1024.5); err != nil {
		t.Fatalf("Failed to record first value: %v", err)
	}
	if err := counter.Record(labels, 2048.75); err != nil {
		t.Fatalf("Failed to record second value: %v", err)
	}

	// Collect metrics and verify counter accumulated the sum
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Find our metric in the collected data
	foundMetric := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test_float_counter" {
				foundMetric = true

				expected := metricdata.Metrics{
					Name:        "test_float_counter",
					Description: "Test float64 counter metric",
					Unit:        "bytes",
					Data: metricdata.Sum[float64]{
						Temporality: metricdata.CumulativeTemporality,
						IsMonotonic: true,
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(
									attribute.String("operation", "read"),
								),
								Value: 3073.25, // 1024.5 + 2048.75
							},
						},
					},
				}

				metricdatatest.AssertEqual(t, expected, m, metricdatatest.IgnoreTimestamp())
			}
		}
	}

	if !foundMetric {
		t.Fatal("test_float_counter metric not found in collected metrics")
	}
}
