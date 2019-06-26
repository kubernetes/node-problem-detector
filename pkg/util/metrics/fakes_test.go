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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFakeInt64Metric(t *testing.T) {
	type recordType struct {
		tags        map[string]string
		measurement int64
	}
	testCases := []struct {
		name            string
		metricName      string
		aggregation     Aggregation
		tagNames        []string
		records         []recordType
		expectedMetrics []Int64MetricRepresentation
	}{
		{
			name:            "empty sum metric",
			metricName:      "foo",
			aggregation:     Sum,
			tagNames:        []string{},
			records:         []recordType{},
			expectedMetrics: []Int64MetricRepresentation{},
		},
		{
			name:        "sum metric with no tag",
			metricName:  "foo",
			aggregation: Sum,
			tagNames:    []string{},
			records: []recordType{
				{
					tags:        map[string]string{},
					measurement: 1,
				},
				{
					tags:        map[string]string{},
					measurement: 2,
				},
			},
			expectedMetrics: []Int64MetricRepresentation{
				{
					Name:   "foo",
					Labels: map[string]string{},
					Value:  3,
				},
			},
		},
		{
			name:        "sum metric with one tag",
			metricName:  "foo",
			aggregation: Sum,
			tagNames:    []string{"A"},
			records: []recordType{
				{
					tags:        map[string]string{"A": "1"},
					measurement: 1,
				},
				{
					tags:        map[string]string{"A": "1"},
					measurement: 2,
				},
			},
			expectedMetrics: []Int64MetricRepresentation{
				{
					Name:   "foo",
					Labels: map[string]string{"A": "1"},
					Value:  3,
				},
			},
		},
		{
			name:        "sum metric with different tags",
			metricName:  "foo",
			aggregation: Sum,
			tagNames:    []string{"A", "B"},
			records: []recordType{
				{
					tags:        map[string]string{"A": "1"},
					measurement: 1,
				},
				{
					tags:        map[string]string{"B": "2"},
					measurement: 2,
				},
				{
					tags:        map[string]string{},
					measurement: 4,
				},
				{
					tags:        map[string]string{"B": "3"},
					measurement: 8,
				},
				{
					tags:        map[string]string{"A": "1"},
					measurement: 16,
				},
			},
			expectedMetrics: []Int64MetricRepresentation{
				{
					Name:   "foo",
					Labels: map[string]string{},
					Value:  4,
				},
				{
					Name:   "foo",
					Labels: map[string]string{"A": "1"},
					Value:  17,
				},
				{
					Name:   "foo",
					Labels: map[string]string{"B": "2"},
					Value:  2,
				},
				{
					Name:   "foo",
					Labels: map[string]string{"B": "3"},
					Value:  8,
				},
			},
		},
		{
			name:            "empty gauge metric",
			metricName:      "foo",
			aggregation:     LastValue,
			tagNames:        []string{},
			records:         []recordType{},
			expectedMetrics: []Int64MetricRepresentation{},
		},
		{
			name:        "gauge metric with one measurement",
			metricName:  "foo",
			aggregation: LastValue,
			tagNames:    []string{},
			records: []recordType{
				{
					tags:        map[string]string{},
					measurement: 2,
				},
			},
			expectedMetrics: []Int64MetricRepresentation{
				{
					Name:   "foo",
					Labels: map[string]string{},
					Value:  2,
				},
			},
		},
		{
			name:        "gauge metric with multiple measurements under same tag",
			metricName:  "foo",
			aggregation: LastValue,
			tagNames:    []string{"A"},
			records: []recordType{
				{
					tags:        map[string]string{"A": "1"},
					measurement: 2,
				},
				{
					tags:        map[string]string{"A": "1"},
					measurement: 4,
				},
			},
			expectedMetrics: []Int64MetricRepresentation{
				{
					Name:   "foo",
					Labels: map[string]string{"A": "1"},
					Value:  4,
				},
			},
		},
		{
			name:        "gauge metric with multiple measurements under different tags",
			metricName:  "foo",
			aggregation: LastValue,
			tagNames:    []string{"A", "B"},
			records: []recordType{
				{
					tags:        map[string]string{"A": "1"},
					measurement: 2,
				},
				{
					tags:        map[string]string{"B": "2"},
					measurement: 4,
				},
				{
					tags:        map[string]string{"A": "1", "B": "2"},
					measurement: 8,
				},
			},
			expectedMetrics: []Int64MetricRepresentation{
				{
					Name:   "foo",
					Labels: map[string]string{"A": "1"},
					Value:  2,
				},
				{
					Name:   "foo",
					Labels: map[string]string{"B": "2"},
					Value:  4,
				},
				{
					Name:   "foo",
					Labels: map[string]string{"A": "1", "B": "2"},
					Value:  8,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			metric := NewFakeInt64Metric(test.metricName, test.aggregation, test.tagNames)

			for _, record := range test.records {
				metric.Record(record.tags, record.measurement)
			}

			gotMetrics := metric.ListMetrics()
			assert.ElementsMatch(t, test.expectedMetrics, gotMetrics,
				"expected metrics: %+v, got: %+v", test.expectedMetrics, gotMetrics)
		})
	}
}
