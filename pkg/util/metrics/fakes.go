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
	"fmt"
	"sync"
)

// FakeInt64Metric is a fake implementation of Int64MetricInterface for testing
type FakeInt64Metric struct {
	name        string
	aggregation Aggregation
	labels      []string
	records     []RecordCall
	mutex       sync.RWMutex
}

// RecordCall represents a call to Record method
type RecordCall struct {
	LabelValues map[string]string
	Value       int64
}

// NewFakeInt64Metric creates a new fake int64 metric
func NewFakeInt64Metric(name string, aggregation Aggregation, labels []string) *FakeInt64Metric {
	return &FakeInt64Metric{
		name:        name,
		aggregation: aggregation,
		labels:      labels,
		records:     make([]RecordCall, 0),
	}
}

// Record implements Int64MetricInterface
func (f *FakeInt64Metric) Record(labelValues map[string]string, value int64) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Copy the labelValues map to avoid reference issues
	copiedLabels := make(map[string]string)
	for k, v := range labelValues {
		copiedLabels[k] = v
	}

	f.records = append(f.records, RecordCall{
		LabelValues: copiedLabels,
		Value:       value,
	})
	return nil
}

// GetRecords returns all recorded calls
func (f *FakeInt64Metric) GetRecords() []RecordCall {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// Return a copy to avoid race conditions
	records := make([]RecordCall, len(f.records))
	copy(records, f.records)
	return records
}

// Reset clears all recorded calls
func (f *FakeInt64Metric) Reset() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.records = f.records[:0]
}

// GetLastValue returns the last recorded value with matching labels
func (f *FakeInt64Metric) GetLastValue(labelValues map[string]string) (int64, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// Search backwards for the last matching record
	for i := len(f.records) - 1; i >= 0; i-- {
		record := f.records[i]
		if mapsEqual(record.LabelValues, labelValues) {
			return record.Value, nil
		}
	}

	return 0, fmt.Errorf("no records found for labels %v", labelValues)
}

// GetTotalValue returns the sum of all recorded values with matching labels
func (f *FakeInt64Metric) GetTotalValue(labelValues map[string]string) int64 {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	var total int64
	for _, record := range f.records {
		if mapsEqual(record.LabelValues, labelValues) {
			total += record.Value
		}
	}

	return total
}

// ListMetrics returns all unique metrics based on aggregation type
func (f *FakeInt64Metric) ListMetrics() []Int64MetricRepresentation {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	if f.aggregation == Sum {
		// For Sum aggregation, aggregate values by labels
		aggregated := make(map[string]int64)
		labelMaps := make(map[string]map[string]string)

		for _, record := range f.records {
			key := labelsMapToString(record.LabelValues)
			aggregated[key] += record.Value
			labelMaps[key] = record.LabelValues
		}

		var metrics []Int64MetricRepresentation
		for key, value := range aggregated {
			metrics = append(metrics, Int64MetricRepresentation{
				Name:   f.name,
				Labels: labelMaps[key],
				Value:  value,
			})
		}

		return metrics
	} else {
		// For LastValue aggregation, return the last value for each unique label set
		lastValues := make(map[string]int64)
		labelMaps := make(map[string]map[string]string)

		for _, record := range f.records {
			key := labelsMapToString(record.LabelValues)
			lastValues[key] = record.Value
			labelMaps[key] = record.LabelValues
		}

		var metrics []Int64MetricRepresentation
		for key, value := range lastValues {
			metrics = append(metrics, Int64MetricRepresentation{
				Name:   f.name,
				Labels: labelMaps[key],
				Value:  value,
			})
		}

		return metrics
	}
}

// FakeFloat64Metric is a fake implementation of Float64MetricInterface for testing
type FakeFloat64Metric struct {
	name        string
	aggregation Aggregation
	labels      []string
	records     []Float64RecordCall
	mutex       sync.RWMutex
}

// Float64RecordCall represents a call to Record method with float64 value
type Float64RecordCall struct {
	LabelValues map[string]string
	Value       float64
}

// NewFakeFloat64Metric creates a new fake float64 metric
func NewFakeFloat64Metric(name string, aggregation Aggregation, labels []string) *FakeFloat64Metric {
	return &FakeFloat64Metric{
		name:        name,
		aggregation: aggregation,
		labels:      labels,
		records:     make([]Float64RecordCall, 0),
	}
}

// Record implements Float64MetricInterface
func (f *FakeFloat64Metric) Record(labelValues map[string]string, value float64) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Copy the labelValues map to avoid reference issues
	copiedLabels := make(map[string]string)
	for k, v := range labelValues {
		copiedLabels[k] = v
	}

	f.records = append(f.records, Float64RecordCall{
		LabelValues: copiedLabels,
		Value:       value,
	})
	return nil
}

// GetRecords returns all recorded calls
func (f *FakeFloat64Metric) GetRecords() []Float64RecordCall {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// Return a copy to avoid race conditions
	records := make([]Float64RecordCall, len(f.records))
	copy(records, f.records)
	return records
}

// Reset clears all recorded calls
func (f *FakeFloat64Metric) Reset() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.records = f.records[:0]
}

// Helper function to compare label maps
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

// Helper function to convert a labels map to a string key for aggregation
func labelsMapToString(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	// Create a consistent string representation of the labels map
	var keys []string
	for k := range labels {
		keys = append(keys, k)
	}

	// Sort keys for consistent ordering
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	result := ""
	for i, k := range keys {
		if i > 0 {
			result += ","
		}
		result += k + "=" + labels[k]
	}

	return result
}
