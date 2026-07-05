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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

// otelMetric wraps an OpenTelemetry instrument for a single numeric type.
// The int64/float64 instrument types differ, so the counter Add and gauge
// Record operations are bound at construction as method values.
type otelMetric[T int64 | float64] struct {
	name        string
	description string
	unit        string
	aggregation Aggregation
	labels      []string
	labelSet    map[string]struct{}
	meter       metric.Meter

	// add is bound to the counter's Add for Sum aggregation; nil otherwise.
	add func(context.Context, T, ...metric.AddOption)
	// record is bound to the gauge's Record for LastValue aggregation; nil otherwise.
	record func(context.Context, T, ...metric.RecordOption)
}

// otelInstrumentFactory constructs the type-specific counter and gauge
// instruments and returns their bound Add/Record method values. Exactly one of
// the returned functions is non-nil, matching the requested aggregation.
type otelInstrumentFactory[T int64 | float64] func(
	meter metric.Meter, name, description, unit string, aggregation Aggregation,
) (add func(context.Context, T, ...metric.AddOption), record func(context.Context, T, ...metric.RecordOption), err error)

// newOTelMetric builds a generic otelMetric, keeping the empty-name,
// aggregation-switch/error, and MetricMap bookkeeping in one place.
func newOTelMetric[T int64 | float64](
	metricID MetricID, name, description, unit string, aggregation Aggregation, labels []string,
	factory otelInstrumentFactory[T],
) (*otelMetric[T], error) {
	if name == "" {
		return nil, nil
	}

	meter := otelutil.GetGlobalMeter()

	labelSet := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		labelSet[label] = struct{}{}
	}

	m := &otelMetric[T]{
		name:        name,
		description: description,
		unit:        unit,
		aggregation: aggregation,
		labels:      labels,
		labelSet:    labelSet,
		meter:       meter,
	}

	switch aggregation {
	case Sum, LastValue:
		add, record, err := factory(meter, name, description, unit, aggregation)
		if err != nil {
			return nil, err
		}
		m.add = add
		m.record = record
	default:
		return nil, fmt.Errorf("unsupported aggregation type for metric %s: %v", name, aggregation)
	}

	// Register metric mapping
	MetricMap.AddMapping(metricID, name)

	return m, nil
}

// Record validates the provided labels against the declared label set and
// dispatches to the counter Add or gauge Record depending on aggregation.
func (m *otelMetric[T]) Record(labelValues map[string]string, value T) error {
	ctx := context.Background()

	// Convert to OTel attributes, rejecting labels that were not declared.
	attrs := make([]attribute.KeyValue, 0, len(labelValues))
	for k, v := range labelValues {
		if _, ok := m.labelSet[k]; !ok {
			return fmt.Errorf("referencing non-existent label %q on metric %q", k, m.name)
		}
		attrs = append(attrs, attribute.String(k, v))
	}

	switch m.aggregation {
	case Sum:
		if m.add != nil {
			m.add(ctx, value, metric.WithAttributes(attrs...))
		}
	case LastValue:
		if m.record != nil {
			m.record(ctx, value, metric.WithAttributes(attrs...))
		}
	default:
		return fmt.Errorf("unsupported aggregation type for metric %s: %v", m.name, m.aggregation)
	}

	return nil
}
