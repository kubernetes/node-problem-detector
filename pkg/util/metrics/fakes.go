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
	"errors"
	"fmt"
	"reflect"
)

// Int64MetricRepresentation represents a snapshot of an int64 metrics.
// This is used for inspecting fake metrics.
type Int64MetricRepresentation struct {
	// Name is the metric name.
	Name string
	// Labels contains all metric labels in key-value pair format.
	Labels map[string]string
	// Value is the value of the metric.
	Value int64
}

// Int64MetricInterface is used to create test double for Int64Metric.
type Int64MetricInterface interface {
	// Record records a measurement for the metric, with provided tags as metric labels.
	Record(tags map[string]string, measurement int64) error
}

// FakeInt64Metric implements Int64MetricInterface.
// FakeInt64Metric can be used as a test double for Int64MetricInterface, allowing
// inspection of the metrics.
type FakeInt64Metric struct {
	name        string
	aggregation Aggregation
	allowedTags map[string]bool
	metrics     []Int64MetricRepresentation
}

func NewFakeInt64Metric(name string, aggregation Aggregation, tagNames []string) *FakeInt64Metric {
	if name == "" {
		return nil
	}

	allowedTags := make(map[string]bool)
	for _, tagName := range tagNames {
		allowedTags[tagName] = true
	}

	fake := FakeInt64Metric{name, aggregation, allowedTags, []Int64MetricRepresentation{}}
	return &fake
}

func (fake *FakeInt64Metric) Record(tags map[string]string, measurement int64) error {
	labels := make(map[string]string)
	for tagName, tagValue := range tags {
		if _, ok := fake.allowedTags[tagName]; !ok {
			return fmt.Errorf("tag %q is not allowed", tagName)
		}
		labels[tagName] = tagValue
	}

	metric := Int64MetricRepresentation{
		Name:   fake.name,
		Labels: labels,
	}

	// If there is a metric with equavalent labels, reuse it.
	metricIndex := -1
	for index, existingMetric := range fake.metrics {
		if !reflect.DeepEqual(existingMetric.Labels, metric.Labels) {
			continue
		}
		metricIndex = index
		break
	}
	// If there is no metric with equalvalent labels, create a new one.
	if metricIndex == -1 {
		fake.metrics = append(fake.metrics, metric)
		metricIndex = len(fake.metrics) - 1
	}

	switch fake.aggregation {
	case LastValue:
		fake.metrics[metricIndex].Value = measurement
	case Sum:
		fake.metrics[metricIndex].Value += measurement
	default:
		return errors.New("unsupported aggregation type")
	}
	return nil
}

// ListMetrics returns a snapshot of the current metrics.
func (fake *FakeInt64Metric) ListMetrics() []Int64MetricRepresentation {
	return fake.metrics
}
