//go:build !disable_stackdriver_exporter

// Copyright 2024 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudmock

import (
	"testing"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertTimeSeriesCount verifies the total number of time series across all requests.
func AssertTimeSeriesCount(t *testing.T, reqs []*monitoringpb.CreateTimeSeriesRequest, expected int) {
	t.Helper()
	total := 0
	for _, req := range reqs {
		total += len(req.TimeSeries)
	}
	assert.Equal(t, expected, total, "unexpected number of time series")
}

// FindTimeSeriesByMetricType searches for a time series with the specified metric type.
// Returns nil if not found.
func FindTimeSeriesByMetricType(reqs []*monitoringpb.CreateTimeSeriesRequest, metricType string) *monitoringpb.TimeSeries {
	for _, req := range reqs {
		for _, ts := range req.TimeSeries {
			if ts.Metric != nil && ts.Metric.Type == metricType {
				return ts
			}
		}
	}
	return nil
}

// AssertMetricExists verifies that a time series with the given metric type exists.
func AssertMetricExists(t *testing.T, reqs []*monitoringpb.CreateTimeSeriesRequest, metricType string) *monitoringpb.TimeSeries {
	t.Helper()
	ts := FindTimeSeriesByMetricType(reqs, metricType)
	require.NotNil(t, ts, "metric type %s not found", metricType)
	return ts
}

// AssertMetricValue verifies that a time series has the expected value.
// Supports int64, float64, and bool values.
func AssertMetricValue(t *testing.T, ts *monitoringpb.TimeSeries, expectedValue interface{}) {
	t.Helper()
	require.NotNil(t, ts, "time series is nil")
	require.NotEmpty(t, ts.Points, "time series has no points")

	point := ts.Points[0]
	require.NotNil(t, point.Value, "point value is nil")

	switch v := expectedValue.(type) {
	case int64:
		assert.Equal(t, v, point.Value.GetInt64Value(), "unexpected int64 value")
	case int:
		assert.Equal(t, int64(v), point.Value.GetInt64Value(), "unexpected int64 value")
	case float64:
		assert.Equal(t, v, point.Value.GetDoubleValue(), "unexpected float64 value")
	case bool:
		assert.Equal(t, v, point.Value.GetBoolValue(), "unexpected bool value")
	default:
		t.Fatalf("unsupported value type: %T", expectedValue)
	}
}

// AssertResourceLabels verifies that a time series has the expected resource labels.
func AssertResourceLabels(t *testing.T, ts *monitoringpb.TimeSeries, expectedLabels map[string]string) {
	t.Helper()
	require.NotNil(t, ts, "time series is nil")
	require.NotNil(t, ts.Resource, "resource is nil")

	for key, expectedValue := range expectedLabels {
		actualValue, exists := ts.Resource.Labels[key]
		require.True(t, exists, "expected resource label %s not found", key)
		assert.Equal(t, expectedValue, actualValue, "unexpected value for resource label %s", key)
	}
}

// AssertMetricLabels verifies that a time series has the expected metric labels.
func AssertMetricLabels(t *testing.T, ts *monitoringpb.TimeSeries, expectedLabels map[string]string) {
	t.Helper()
	require.NotNil(t, ts, "time series is nil")
	require.NotNil(t, ts.Metric, "metric is nil")

	for key, expectedValue := range expectedLabels {
		actualValue, exists := ts.Metric.Labels[key]
		require.True(t, exists, "expected metric label %s not found", key)
		assert.Equal(t, expectedValue, actualValue, "unexpected value for metric label %s", key)
	}
}
