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

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
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
type Int64Metric struct {
	name    string
	measure *stats.Int64Measure
}

// NewInt64Metric create a Int64Metric metric, returns nil when viewName is empty.
func NewInt64Metric(metricID MetricID, viewName string, description string, unit string, aggregation Aggregation, tagNames []string) (*Int64Metric, error) {
	if viewName == "" {
		return nil, nil
	}

	MetricMap.AddMapping(metricID, viewName)

	tagKeys, err := getTagKeysFromNames(tagNames)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric %q because of tag creation failure: %v", viewName, err)
	}

	var aggregationMethod *view.Aggregation
	switch aggregation {
	case LastValue:
		aggregationMethod = view.LastValue()
	case Sum:
		aggregationMethod = view.Sum()
	default:
		return nil, fmt.Errorf("unknown aggregation option %q", aggregation)
	}

	measure := stats.Int64(viewName, description, unit)
	newView := &view.View{
		Name:        viewName,
		Measure:     measure,
		Description: description,
		Aggregation: aggregationMethod,
		TagKeys:     tagKeys,
	}
	view.Register(newView)

	metric := Int64Metric{viewName, measure}
	return &metric, nil
}

// Record records a measurement for the metric, with provided tags as metric labels.
func (metric *Int64Metric) Record(tags map[string]string, measurement int64) error {
	var mutators []tag.Mutator

	tagMapMutex.RLock()
	defer tagMapMutex.RUnlock()

	for tagName, tagValue := range tags {
		tagKey, ok := tagMap[tagName]
		if !ok {
			return fmt.Errorf("referencing none existing tag %q in metric %q", tagName, metric.name)
		}
		mutators = append(mutators, tag.Upsert(tagKey, tagValue))
	}

	return stats.RecordWithTags(
		context.Background(),
		mutators,
		metric.measure.M(measurement))
}
