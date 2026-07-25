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
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

// otelMetric wraps an OpenTelemetry instrument for a single numeric type.
// The int64/float64 instrument types differ, so the measurement operation is
// bound at construction.
type otelMetric[T int64 | float64] struct {
	name     string
	labelSet map[string]struct{}

	// emit sends one measurement to the underlying counter (Sum) or
	// gauge (LastValue) instrument.
	emit func(context.Context, T, metric.MeasurementOption)
}

// otelInstrumentFactory constructs the type-specific counter or gauge
// instrument for the given aggregation and returns its bound measurement
// operation.
type otelInstrumentFactory[T int64 | float64] func(
	meter metric.Meter, name, description, unit string, aggregation Aggregation,
) (emit func(context.Context, T, metric.MeasurementOption), err error)

// newOTelMetric builds a generic otelMetric, keeping the empty-name,
// aggregation-switch/error, and MetricMap bookkeeping in one place.
func newOTelMetric[T int64 | float64](
	metricID MetricID, name, description, unit string, aggregation Aggregation, labels []string,
	factory otelInstrumentFactory[T],
) (*otelMetric[T], error) {
	if name == "" {
		return nil, nil
	}

	switch aggregation {
	case Sum, LastValue:
	default:
		return nil, fmt.Errorf("unsupported aggregation type for metric %s: %v", name, aggregation)
	}

	instrumentName := normalizeMetricName(name)
	emit, err := factory(otelutil.GetGlobalMeter(), instrumentName, description, unit, aggregation)
	if err != nil {
		return nil, err
	}

	labelSet := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		labelSet[label] = struct{}{}
	}

	// Register metric mapping
	MetricMap.AddMapping(metricID, instrumentName)

	return &otelMetric[T]{name: instrumentName, labelSet: labelSet, emit: emit}, nil
}

// Record validates the provided labels against the declared label set and
// emits the measurement to the underlying instrument.
func (m *otelMetric[T]) Record(labelValues map[string]string, value T) error {
	attrs := make([]attribute.KeyValue, 0, len(labelValues))
	for k, v := range labelValues {
		if _, ok := m.labelSet[k]; !ok {
			return fmt.Errorf("referencing non-existent label %q on metric %q", k, m.name)
		}
		attrs = append(attrs, attribute.String(k, v))
	}

	m.emit(context.Background(), value, metric.WithAttributes(attrs...))
	return nil
}

// normalizeMetricName preserves the naming behavior of the legacy OpenCensus
// Prometheus exporter while producing a valid OpenTelemetry instrument name.
func normalizeMetricName(name string) string {
	const legacyNameLimit = 100

	if name == "" {
		return ""
	}
	if len(name) > legacyNameLimit {
		name = name[:legacyNameLimit]
	}

	var normalized strings.Builder
	normalized.Grow(len(name) + len("key"))
	if name[0] >= '0' && name[0] <= '9' {
		normalized.WriteByte('_')
	}

	var previous rune
	for _, current := range name {
		switch {
		case current >= 'a' && current <= 'z',
			current >= 'A' && current <= 'Z',
			current >= '0' && current <= '9',
			current == '_':
			normalized.WriteRune(current)
		case current == '-' && previous == '-':
		default:
			normalized.WriteByte('_')
		}
		previous = current
	}

	result := normalized.String()
	if result[0] == '_' {
		return "key" + result
	}
	return result
}
