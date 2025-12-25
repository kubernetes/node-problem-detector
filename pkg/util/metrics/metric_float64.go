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
	otelmetric "go.opentelemetry.io/otel/metric"
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

// Float64Metric represents an float64 metric.
type Float64Metric struct {
	name        string
	aggregation Aggregation
	counter     otelmetric.Float64Counter
	gauge       otelmetric.Float64Gauge
	attrKeys    []attribute.Key
}

// NewFloat64Metric create a Float64Metric metrics, returns nil when viewName is empty.
func NewFloat64Metric(metricID MetricID, viewName string, description string, unit string, aggregation Aggregation, tagNames []string) (*Float64Metric, error) {
	if viewName == "" {
		return nil, nil
	}

	MetricMap.AddMapping(metricID, viewName)

	attrKeys := getAttributeKeysFromNames(tagNames)

	m := GetMeter()
	fm := &Float64Metric{
		name:        viewName,
		aggregation: aggregation,
		attrKeys:    attrKeys,
	}

	var err error
	switch aggregation {
	case LastValue:
		fm.gauge, err = m.Float64Gauge(viewName,
			otelmetric.WithDescription(description),
			otelmetric.WithUnit(unit))
		if err != nil {
			return nil, fmt.Errorf("failed to create gauge metric %q: %v", viewName, err)
		}
	case Sum:
		fm.counter, err = m.Float64Counter(viewName,
			otelmetric.WithDescription(description),
			otelmetric.WithUnit(unit))
		if err != nil {
			return nil, fmt.Errorf("failed to create counter metric %q: %v", viewName, err)
		}
	default:
		return nil, fmt.Errorf("unknown aggregation option %q", aggregation)
	}

	return fm, nil
}

// Record records a measurement for the metric, with provided tags as metric labels.
func (m *Float64Metric) Record(tags map[string]string, measurement float64) error {
	attributeKeyMapMutex.RLock()
	defer attributeKeyMapMutex.RUnlock()

	attrs := make([]attribute.KeyValue, 0, len(tags))
	for tagName, tagValue := range tags {
		attrKey, ok := attributeKeyMap[tagName]
		if !ok {
			return fmt.Errorf("referencing none existing tag %q in metric %q", tagName, m.name)
		}
		attrs = append(attrs, attrKey.String(tagValue))
	}

	ctx := context.Background()
	opt := otelmetric.WithAttributes(attrs...)

	switch m.aggregation {
	case LastValue:
		m.gauge.Record(ctx, measurement, opt)
	case Sum:
		m.counter.Add(ctx, measurement, opt)
	}

	return nil
}
