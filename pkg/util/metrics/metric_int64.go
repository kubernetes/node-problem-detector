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
package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/klog/v2"

	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

// Int64MetricRepresentation represents a snapshot of an int64 metrics.
// This is used for inspecting metric internals.
type Int64MetricRepresentation struct {
	// Name is the metric name.
	Name string
	// Labels contains all metric labels in key-value pair format.
	Labels map[string]string
	// Value is the value of the metric.
	Value int64
}

// Int64Metric represents an int64 metric.
// Type alias added for backward compatibility
type Int64Metric = OTelInt64Metric

type Int64MetricInterface interface {
	Record(labelValues map[string]string, value int64) error
}

// NewInt64Metric creates a new Int64 metric using OpenTelemetry
func NewInt64Metric(metricID MetricID, name, description, unit string, aggregation Aggregation, labels []string) (*Int64Metric, error) {
	meter := otelutil.GetGlobalMeter()

	otelMetric := &OTelInt64Metric{
		name:        name,
		description: description,
		unit:        unit,
		aggregation: aggregation,
		labels:      labels,
		meter:       meter,
	}

	var err error
	switch aggregation {
	case Sum:
		otelMetric.counter, err = meter.Int64Counter(
			name,
			metric.WithDescription(description),
			metric.WithUnit(unit),
		)
	case LastValue:
		// Use synchronous Int64Gauge for proper gauge semantics without automatic suffixing
		otelMetric.gauge, err = meter.Int64Gauge(
			name,
			metric.WithDescription(description),
			metric.WithUnit(unit),
		)
	default:
		klog.Warningf("Unsupported aggregation type for metric %s: %v", name, aggregation)
	}

	if err != nil {
		return nil, err
	}

	// Register metric mapping
	MetricMap.AddMapping(metricID, name)

	return otelMetric, nil
}

// OTelInt64Metric wraps OpenTelemetry int64 instruments
type OTelInt64Metric struct {
	name        string
	description string
	unit        string
	aggregation Aggregation
	labels      []string
	counter     metric.Int64Counter
	gauge       metric.Int64Gauge
	meter       metric.Meter
}

// Record implements Int64MetricInterface
func (m *OTelInt64Metric) Record(labelValues map[string]string, value int64) error {
	ctx := context.Background()

	// Convert to OTel attributes
	attrs := make([]attribute.KeyValue, 0, len(labelValues))
	for k, v := range labelValues {
		attrs = append(attrs, attribute.String(k, v))
	}

	switch m.aggregation {
	case Sum:
		if m.counter != nil {
			m.counter.Add(ctx, value, metric.WithAttributes(attrs...))
		}
	case LastValue:
		if m.gauge != nil {
			// For synchronous gauge, directly record the value
			m.gauge.Record(ctx, value, metric.WithAttributes(attrs...))
		}
	default:
		klog.Warningf("Unsupported aggregation type: %v", m.aggregation)
	}

	return nil
}
