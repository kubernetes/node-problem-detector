//go:build !disable_stackdriver_exporter

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

package stackdriverexporter

import (
	"context"
	"testing"
	"time"

	gcpmetric "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"k8s.io/node-problem-detector/pkg/exporters"
	seconfig "k8s.io/node-problem-detector/pkg/exporters/stackdriver/config"
	"k8s.io/node-problem-detector/pkg/exporters/stackdriver/internal/cloudmock"
	"k8s.io/node-problem-detector/pkg/util/metrics"
	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

func TestRegistration(t *testing.T) {
	assert.NotPanics(t,
		func() { exporters.GetExporterHandlerOrDie(exporterName) },
		"Stackdriver exporter failed to register itself as an exporter.")
}

func TestMetricTypeConversion(t *testing.T) {
	// Set up metric mappings (normally done when metrics are created)
	metrics.MetricMap.AddMapping(metrics.HostUptimeID, "host/uptime")
	metrics.MetricMap.AddMapping(metrics.CPULoad1m, "cpu/load_1m")
	metrics.MetricMap.AddMapping(metrics.MemoryBytesUsedID, "memory/bytes_used")

	tests := []struct {
		metricName   string
		expectedType string
	}{
		{
			metricName:   "host/uptime",
			expectedType: "compute.googleapis.com/guest/system/uptime",
		},
		{
			metricName:   "cpu/load_1m",
			expectedType: "compute.googleapis.com/guest/cpu/load_1m",
		},
		{
			metricName:   "memory/bytes_used",
			expectedType: "compute.googleapis.com/guest/memory/bytes_used",
		},
		{
			metricName:   "unknown/metric",
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.metricName, func(t *testing.T) {
			converter := getMetricTypeConversionFunction("")
			result := converter(tt.metricName)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestCustomMetricPrefix(t *testing.T) {
	// Set up metric mappings
	metrics.MetricMap.AddMapping(metrics.HostUptimeID, "host/uptime")

	tests := []struct {
		metricName   string
		expectedType string
		description  string
	}{
		{
			metricName:   "host/uptime",
			expectedType: "compute.googleapis.com/guest/system/uptime",
			description:  "known metric should ignore custom prefix",
		},
		{
			metricName:   "custom/my_metric",
			expectedType: "custom.googleapis.com/npd/custom/my_metric",
			description:  "unknown metric should use custom prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			converter := getMetricTypeConversionFunction("custom.googleapis.com/npd")
			result := converter(tt.metricName)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestExportMetricsToCloudMonitoring(t *testing.T) {
	// Reset global OTel state for clean test
	otelutil.ResetForTesting()

	// Create mock Google Cloud Monitoring server
	mockServer := cloudmock.NewMetricTestServer()
	defer mockServer.Shutdown()

	// Create GCP exporter configured to use mock server
	exporter, err := gcpmetric.New(
		gcpmetric.WithProjectID("test-project-dy"),
		gcpmetric.WithMetricDescriptorTypeFormatter(func(m metricdata.Metrics) string {
			// Use our NPD metric conversion function
			converter := getMetricTypeConversionFunction("")
			return converter(m.Name)
		}),
		gcpmetric.WithMonitoringClientOptions(
			option.WithEndpoint(mockServer.Endpoint()),
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		),
	)
	require.NoError(t, err)

	// Create a manual reader for deterministic export
	reader := sdkmetric.NewManualReader()

	// Create meter provider with both readers
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(100*time.Millisecond))),
	)
	otel.SetMeterProvider(provider)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	// Register NPD metrics mappings
	metrics.MetricMap.AddMapping(metrics.HostUptimeID, "host/uptime")
	metrics.MetricMap.AddMapping(metrics.CPULoad1m, "cpu/load_1m")
	metrics.MetricMap.AddMapping(metrics.MemoryBytesUsedID, "memory/bytes_used")

	// Create and record some NPD metrics
	uptimeMetric, err := metrics.NewInt64Metric(
		metrics.HostUptimeID,
		"host/uptime",
		"System uptime in seconds",
		"s",
		metrics.LastValue,
		[]string{},
	)
	require.NoError(t, err)

	err = uptimeMetric.Record(map[string]string{}, int64(12345))
	require.NoError(t, err)

	cpuLoadMetric, err := metrics.NewFloat64Metric(
		metrics.CPULoad1m,
		"cpu/load_1m",
		"CPU load average 1 minute",
		"1",
		metrics.LastValue,
		[]string{"cpu"},
	)
	require.NoError(t, err)

	err = cpuLoadMetric.Record(map[string]string{"cpu": "0"}, 1.5)
	require.NoError(t, err)

	memoryMetric, err := metrics.NewInt64Metric(
		metrics.MemoryBytesUsedID,
		"memory/bytes_used",
		"Memory bytes used",
		"By",
		metrics.LastValue,
		[]string{},
	)
	require.NoError(t, err)

	err = memoryMetric.Record(map[string]string{}, int64(1073741824))
	require.NoError(t, err)

	// Wait for periodic export to happen
	time.Sleep(300 * time.Millisecond)

	// Force flush to ensure all metrics are exported
	err = exporter.ForceFlush(context.Background())
	require.NoError(t, err)

	// Verify metrics were received by mock server
	reqs := mockServer.CreateTimeSeriesRequests()
	require.NotEmpty(t, reqs, "should have received metric requests from exporter")

	// Count total time series across all requests
	totalTimeSeries := 0
	for _, req := range reqs {
		totalTimeSeries += len(req.TimeSeries)
	}
	require.Greater(t, totalTimeSeries, 0, "should have exported at least one metric")

	// Verify metric types were converted correctly
	foundUptime := false
	foundCPULoad := false
	foundMemory := false

	for _, req := range reqs {
		for _, ts := range req.TimeSeries {
			switch ts.Metric.Type {
			case "compute.googleapis.com/guest/system/uptime":
				foundUptime = true
				// Verify the value
				require.NotEmpty(t, ts.Points)
				assert.Equal(t, int64(12345), ts.Points[0].Value.GetInt64Value())
			case "compute.googleapis.com/guest/cpu/load_1m":
				foundCPULoad = true
				require.NotEmpty(t, ts.Points)
				assert.InDelta(t, 1.5, ts.Points[0].Value.GetDoubleValue(), 0.01)
			case "compute.googleapis.com/guest/memory/bytes_used":
				foundMemory = true
				require.NotEmpty(t, ts.Points)
				assert.Equal(t, int64(1073741824), ts.Points[0].Value.GetInt64Value())
			}
		}
	}

	assert.True(t, foundUptime, "should have exported host uptime metric")
	assert.True(t, foundCPULoad, "should have exported CPU load metric")
	assert.True(t, foundMemory, "should have exported memory bytes metric")

	t.Logf("Successfully exported and verified %d time series to mock GCM", totalTimeSeries)
}

func TestConfigurationApplyDefaults(t *testing.T) {
	config := seconfig.StackdriverExporterConfig{}
	config.ApplyConfiguration()

	assert.Equal(t, "1m0s", config.ExportPeriod)
	assert.Equal(t, "monitoring.googleapis.com:443", config.APIEndpoint)
	assert.Equal(t, "10m0s", config.MetadataFetchTimeout)
	assert.Equal(t, "10s", config.MetadataFetchInterval)
}
