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
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

func TestFloat64GaugeSetValueSemantics(t *testing.T) {
	// Set up SDK with ManualReader for testing
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(otelutil.GetResource()),
		sdkmetric.WithReader(reader),
	)
	meter := provider.Meter("test")

	// Create a gauge metric
	gauge, err := meter.Float64Gauge("test_float_gauge",
		metric.WithDescription("Test float64 gauge metric"),
		metric.WithUnit("percent"),
	)
	if err != nil {
		t.Fatalf("Failed to create gauge metric: %v", err)
	}

	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("component", "cpu"),
		attribute.String("state", "usage"),
	}

	// Set initial value to 0.0 (initialization)
	gauge.Record(ctx, 0.0, metric.WithAttributes(attrs...))

	// Set value to 42.5 (current reading)
	gauge.Record(ctx, 42.5, metric.WithAttributes(attrs...))

	// Collect metrics and verify gauge shows the last value (42.5)
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	expected := metricdata.Metrics{
		Name:        "test_float_gauge",
		Description: "Test float64 gauge metric",
		Unit:        "percent",
		Data: metricdata.Gauge[float64]{
			DataPoints: []metricdata.DataPoint[float64]{
				{
					Attributes: attribute.NewSet(attrs...),
					Value:      42.5,
				},
			},
		},
	}

	metricdatatest.AssertEqual(t, expected, rm.ScopeMetrics[0].Metrics[0], metricdatatest.IgnoreTimestamp())

	// Set value to 15.3 (new reading)
	gauge.Record(ctx, 15.3, metric.WithAttributes(attrs...))

	// Collect again and verify gauge now shows 15.3
	rm = metricdata.ResourceMetrics{}
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	expected.Data = metricdata.Gauge[float64]{
		DataPoints: []metricdata.DataPoint[float64]{
			{
				Attributes: attribute.NewSet(attrs...),
				Value:      15.3,
			},
		},
	}

	metricdatatest.AssertEqual(t, expected, rm.ScopeMetrics[0].Metrics[0], metricdatatest.IgnoreTimestamp())
}

func TestFloat64CounterAddSemantics(t *testing.T) {
	// Set up SDK with ManualReader for testing
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(otelutil.GetResource()),
		sdkmetric.WithReader(reader),
	)
	meter := provider.Meter("test")

	// Create a counter metric
	counter, err := meter.Float64Counter("test_float_counter",
		metric.WithDescription("Test float64 counter metric"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		t.Fatalf("Failed to create counter metric: %v", err)
	}

	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("operation", "read"),
	}

	// Add to counter multiple times with fractional values
	counter.Add(ctx, 1024.5, metric.WithAttributes(attrs...))
	counter.Add(ctx, 2048.75, metric.WithAttributes(attrs...))

	// Collect metrics and verify counter accumulated the sum
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}

	expected := metricdata.Metrics{
		Name:        "test_float_counter",
		Description: "Test float64 counter metric",
		Unit:        "bytes",
		Data: metricdata.Sum[float64]{
			Temporality: metricdata.CumulativeTemporality,
			IsMonotonic: true,
			DataPoints: []metricdata.DataPoint[float64]{
				{
					Attributes: attribute.NewSet(attrs...),
					Value:      3073.25, // 1024.5 + 2048.75
				},
			},
		},
	}

	metricdatatest.AssertEqual(t, expected, rm.ScopeMetrics[0].Metrics[0], metricdatatest.IgnoreTimestamp())
}
