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

	"go.opentelemetry.io/otel/metric"
)

// Float64MetricRepresentation represents a snapshot of a float64 metrics.
// This is used for inspecting metric internals.
type Float64MetricRepresentation struct {
	// Name is the metric name.
	Name string
	// Labels contains all metric labels in key-value pair format.
	Labels map[string]string
	// Value is the value of the metric.
	Value float64
}

// OTelFloat64Metric wraps OpenTelemetry float64 instruments.
type OTelFloat64Metric = otelMetric[float64]

// Float64Metric represents an float64 metric.
// Type alias added for backward compatibility
type Float64Metric = OTelFloat64Metric

type Float64MetricInterface interface {
	Record(labelValues map[string]string, value float64) error
}

// NewFloat64Metric creates a new Float64 metric using OpenTelemetry, returns nil when name is empty
func NewFloat64Metric(metricID MetricID, name, description, unit string, aggregation Aggregation, labels []string) (*Float64Metric, error) {
	return newOTelMetric(metricID, name, description, unit, aggregation, labels, newFloat64Instrument)
}

// newFloat64Instrument constructs the float64 counter/gauge for the given aggregation.
func newFloat64Instrument(
	meter metric.Meter, name, description, unit string, aggregation Aggregation,
) (func(context.Context, float64, metric.MeasurementOption), error) {
	switch aggregation {
	case Sum:
		counter, err := meter.Float64Counter(name,
			metric.WithDescription(description),
			metric.WithUnit(unit),
		)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, value float64, opt metric.MeasurementOption) {
			counter.Add(ctx, value, opt)
		}, nil
	default: // LastValue
		gauge, err := meter.Float64Gauge(name,
			metric.WithDescription(description),
			metric.WithUnit(unit),
		)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context, value float64, opt metric.MeasurementOption) {
			gauge.Record(ctx, value, opt)
		}, nil
	}
}
