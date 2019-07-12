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
	"sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var tagMap map[string]tag.Key
var tagMapMutex sync.RWMutex

func init() {
	tagMapMutex.Lock()
	tagMap = make(map[string]tag.Key)
	tagMapMutex.Unlock()
}

// Int64Metric represents an int64 metric.
type Int64Metric struct {
	name    string
	measure *stats.Int64Measure
}

// Aggregation defines how measurements should be aggregated into data points.
type Aggregation string

const (
	// LastValue means last measurement overwrites previous measurements (gauge metric).
	LastValue Aggregation = "LastValue"
	// Sum means last measurement will be added onto previous measurements (counter metric).
	Sum Aggregation = "Sum"
)

// NewInt64Metric create a Int64Metric metric, returns nil when name is empty.
func NewInt64Metric(name string, description string, unit string, aggregation Aggregation, tagNames []string) (*Int64Metric, error) {
	if name == "" {
		return nil, nil
	}

	tagKeys, err := getTagKeysFromNames(tagNames)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric %q because of tag creation failure: %v", name, err)
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

	measure := stats.Int64(name, description, unit)
	newView := &view.View{
		Name:        name,
		Measure:     measure,
		Description: description,
		Aggregation: aggregationMethod,
		TagKeys:     tagKeys,
	}
	view.Register(newView)

	metric := Int64Metric{name, measure}
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

// Float64Metric represents an float64 metric.
type Float64Metric struct {
	name    string
	measure *stats.Float64Measure
}

// NewFloat64Metric create a Float64Metric metrics, returns nil when name is empty.
func NewFloat64Metric(name string, description string, unit string, aggregation Aggregation, tagNames []string) (*Float64Metric, error) {
	if name == "" {
		return nil, nil
	}

	tagKeys, err := getTagKeysFromNames(tagNames)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric %q because of tag creation failure: %v", name, err)
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

	measure := stats.Float64(name, description, unit)
	newView := &view.View{
		Name:        name,
		Measure:     measure,
		Description: description,
		Aggregation: aggregationMethod,
		TagKeys:     tagKeys,
	}
	view.Register(newView)

	metric := Float64Metric{name, measure}
	return &metric, nil
}

// Record records a measurement for the metric, with provided tags as metric labels.
func (metric *Float64Metric) Record(tags map[string]string, measurement float64) error {
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

func getTagKeysFromNames(tagNames []string) ([]tag.Key, error) {
	tagMapMutex.Lock()
	defer tagMapMutex.Unlock()

	var tagKeys []tag.Key
	var err error
	for _, tagName := range tagNames {
		tagKey, ok := tagMap[tagName]
		if !ok {
			tagKey, err = tag.NewKey(tagName)
			if err != nil {
				return []tag.Key{}, fmt.Errorf("failed to create tag %q: %v", tagName, err)
			}
			tagMap[tagName] = tagKey
		}
		tagKeys = append(tagKeys, tagKey)
	}
	return tagKeys, nil
}
