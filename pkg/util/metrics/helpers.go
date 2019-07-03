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
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// NewInt64Metric create a stats.Int64 metrics, returns nil when name is empty.
func NewInt64Metric(name string, description string, unit string, aggregation *view.Aggregation, tagKeys []tag.Key) *stats.Int64Measure {
	if name == "" {
		return nil
	}
	measure := stats.Int64(name, description, unit)
	newView := &view.View{
		Name:        name,
		Measure:     measure,
		Description: description,
		Aggregation: aggregation,
		TagKeys:     tagKeys,
	}
	view.Register(newView)
	return measure
}

// NewFloat64Metric create a stats.Float64 metrics, returns nil when name is empty.
func NewFloat64Metric(name string, description string, unit string, aggregation *view.Aggregation, tagKeys []tag.Key) *stats.Float64Measure {
	if name == "" {
		return nil
	}
	measure := stats.Float64(name, description, unit)
	newView := &view.View{
		Name:        name,
		Measure:     measure,
		Description: description,
		Aggregation: aggregation,
		TagKeys:     tagKeys,
	}
	view.Register(newView)
	return measure
}
