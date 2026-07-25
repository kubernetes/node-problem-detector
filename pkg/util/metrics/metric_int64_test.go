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
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

func TestGaugeSetValueSemantics(t *testing.T) {
	// Reset global state for isolated testing
	otelutil.ResetForTesting()

	// Set up SDK with ManualReader for testing our metrics
	reader := sdkmetric.NewManualReader()

	// Register reader with our global meter provider for our metrics to use
	otelutil.AddMetricReader(reader)
	otelutil.InitializeMeterProvider()

	// Create a gauge using our actual NewInt64Metric function
	gauge, err := NewInt64Metric(
		"test_gauge",
		"test_gauge",
		"Test gauge metric",
		"1",
		LastValue,
		[]string{"reason", "type"},
	)
	if err != nil {
		t.Fatalf("Failed to create gauge metric: %v", err)
	}

	ctx := context.Background()
	labels1 := map[string]string{
		"reason": "TestReason",
		"type":   "TestType",
	}

	// Set initial value to 0 (initialization)
	if err := gauge.Record(labels1, 0); err != nil {
		t.Fatalf("Failed to record initial value: %v", err)
	}

	// Set value to 1 (problem detected)
	if err := gauge.Record(labels1, 1); err != nil {
		t.Fatalf("Failed to record updated value: %v", err)
	}

	// Collect metrics and verify gauge shows the last value (1)
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	// Find our metric in the collected data
	foundMetric := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test_gauge" {
				foundMetric = true

				expected := metricdata.Metrics{
					Name:        "test_gauge",
					Description: "Test gauge metric",
					Unit:        "1",
					Data: metricdata.Gauge[int64]{
						DataPoints: []metricdata.DataPoint[int64]{
							{
								Attributes: attribute.NewSet(
									attribute.String("reason", "TestReason"),
									attribute.String("type", "TestType"),
								),
								Value: 1,
							},
						},
					},
				}

				metricdatatest.AssertEqual(t, expected, m, metricdatatest.IgnoreTimestamp())
			}
		}
	}

	if !foundMetric {
		t.Fatal("test_gauge metric not found in collected metrics")
	}

	// Set value back to 0 (problem resolved)
	if err := gauge.Record(labels1, 0); err != nil {
		t.Fatalf("Failed to record resolved value: %v", err)
	}

	// Collect again and verify gauge now shows 0
	rm = metricdata.ResourceMetrics{}
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	foundMetric = false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test_gauge" {
				foundMetric = true

				expected := metricdata.Metrics{
					Name:        "test_gauge",
					Description: "Test gauge metric",
					Unit:        "1",
					Data: metricdata.Gauge[int64]{
						DataPoints: []metricdata.DataPoint[int64]{
							{
								Attributes: attribute.NewSet(
									attribute.String("reason", "TestReason"),
									attribute.String("type", "TestType"),
								),
								Value: 0,
							},
						},
					},
				}

				metricdatatest.AssertEqual(t, expected, m, metricdatatest.IgnoreTimestamp())
			}
		}
	}

	if !foundMetric {
		t.Fatal("test_gauge metric not found in second collection")
	}
}

func TestCounterAddSemantics(t *testing.T) {
	// Reset global state for isolated testing
	otelutil.ResetForTesting()

	// Set up SDK with ManualReader for testing our metrics
	reader := sdkmetric.NewManualReader()

	// Register reader with our global meter provider for our metrics to use
	otelutil.AddMetricReader(reader)
	otelutil.InitializeMeterProvider()

	// Create a counter using our actual NewInt64Metric function
	counter, err := NewInt64Metric(
		"test_counter",
		"test_counter",
		"Test counter metric",
		"1",
		Sum,
		[]string{"reason"},
	)
	if err != nil {
		t.Fatalf("Failed to create counter metric: %v", err)
	}

	ctx := context.Background()
	labels := map[string]string{
		"reason": "TestReason",
	}

	// Add to counter twice using our Record method
	if err := counter.Record(labels, 5); err != nil {
		t.Fatalf("Failed to record first value: %v", err)
	}
	if err := counter.Record(labels, 3); err != nil {
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
			if m.Name == "test_counter" {
				foundMetric = true

				expected := metricdata.Metrics{
					Name:        "test_counter",
					Description: "Test counter metric",
					Unit:        "1",
					Data: metricdata.Sum[int64]{
						Temporality: metricdata.CumulativeTemporality,
						IsMonotonic: true,
						DataPoints: []metricdata.DataPoint[int64]{
							{
								Attributes: attribute.NewSet(
									attribute.String("reason", "TestReason"),
								),
								Value: 8, // 5 + 3
							},
						},
					},
				}

				metricdatatest.AssertEqual(t, expected, m, metricdatatest.IgnoreTimestamp())
			}
		}
	}

	if !foundMetric {
		t.Fatal("test_counter metric not found in collected metrics")
	}
}

func TestNewInt64MetricUnsupportedAggregation(t *testing.T) {
	otelutil.ResetForTesting()
	otelutil.AddMetricReader(sdkmetric.NewManualReader())
	otelutil.InitializeMeterProvider()

	// An unsupported aggregation must return an error, not a nil-instrument metric.
	if _, err := NewInt64Metric("bad_agg", "bad_agg", "desc", "1", Aggregation("bogus"), []string{"reason"}); err == nil {
		t.Fatal("expected error for unsupported aggregation type, got nil")
	}

	// The zero value of Aggregation must not silently mean a valid aggregation.
	var zero Aggregation
	if _, err := NewInt64Metric("zero_agg", "zero_agg", "desc", "1", zero, []string{"reason"}); err == nil {
		t.Fatal("expected error for zero-value aggregation type, got nil")
	}
}

func TestInt64MetricRecordUndeclaredLabel(t *testing.T) {
	otelutil.ResetForTesting()
	reader := sdkmetric.NewManualReader()
	otelutil.AddMetricReader(reader)
	otelutil.InitializeMeterProvider()

	metric, err := NewInt64Metric("labeled_metric", "labeled_metric", "desc", "1", Sum, []string{"reason", "value"})
	if err != nil {
		t.Fatalf("Failed to create metric: %v", err)
	}

	// Declared label records fine.
	if err := metric.Record(map[string]string{"reason": "TestReason"}, 1); err != nil {
		t.Fatalf("expected declared label to record, got error: %v", err)
	}

	// Undeclared label key must be rejected.
	if err := metric.Record(map[string]string{"typo": "oops"}, 1); err == nil {
		t.Fatal("expected error for undeclared label key, got nil")
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "labeled_metric" {
				continue
			}
			points := m.Data.(metricdata.Sum[int64]).DataPoints
			if got, ok := points[0].Attributes.Value("value"); !ok || got.AsString() != "" {
				t.Fatalf("expected omitted declared label to be emitted as empty, got %v, %t", got, ok)
			}
			return
		}
	}
	t.Fatal("labeled_metric metric not found")
}

func TestNormalizeMetricName(t *testing.T) {
	longName := strings.Repeat("a", 101)
	tests := []struct {
		name string
		want string
	}{
		{name: "", want: ""},
		{name: "host/uptime", want: "host_uptime"},
		{name: "node:uptime", want: "node_uptime"},
		{name: "disk//used", want: "disk__used"},
		{name: "disk--used", want: "disk_used"},
		{name: "1st metric", want: "key_1st_metric"},
		{name: "_private", want: "key_private"},
		{name: longName, want: strings.Repeat("a", 100)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeMetricName(test.name); got != test.want {
				t.Fatalf("normalizeMetricName(%q) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}
