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

// OTelInt64Metric wraps OpenTelemetry int64 instruments.
type OTelInt64Metric = otelMetric[int64]

// Int64Metric represents an int64 metric.
// Type alias added for backward compatibility
type Int64Metric = OTelInt64Metric

type Int64MetricInterface interface {
	Record(labelValues map[string]string, value int64) error
}

// NewInt64Metric creates a new Int64 metric using OpenTelemetry, returns nil when name is empty
func NewInt64Metric(metricID MetricID, name, description, unit string, aggregation Aggregation, labels []string) (*Int64Metric, error) {
	return newOTelMetric(metricID, name, description, unit, aggregation, labels, newInt64Instrument)
}

// newInt64Instrument constructs the int64 counter/gauge for the given aggregation.
func newInt64Instrument(
	meter metric.Meter, name, description, unit string, aggregation Aggregation,
) (add func(context.Context, int64, ...metric.AddOption), record func(context.Context, int64, ...metric.RecordOption), err error) {
	switch aggregation {
	case Sum:
		counter, cErr := meter.Int64Counter(name,
			metric.WithDescription(description),
			metric.WithUnit(unit),
		)
		if cErr != nil {
			return nil, nil, cErr
		}
		return counter.Add, nil, nil
	default: // LastValue
		// Use synchronous Int64Gauge for proper gauge semantics without automatic suffixing
		gauge, gErr := meter.Int64Gauge(name,
			metric.WithDescription(description),
			metric.WithUnit(unit),
		)
		if gErr != nil {
			return nil, nil, gErr
		}
		return nil, gauge.Record, nil
	}
}
