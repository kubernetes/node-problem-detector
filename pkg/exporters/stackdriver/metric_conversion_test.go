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

package stackdriverexporter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/node-problem-detector/pkg/util/metrics"
)

// TestNPDMetricToSDMetric verifies that all entries in the NPDMetricToSDMetric map
// are valid and properly formatted.
func TestNPDMetricToSDMetric(t *testing.T) {
	tests := []struct {
		metricID           metrics.MetricID
		expectedSDMetric   string
		expectedPrefix     string
	}{
		// CPU metrics
		{metrics.CPURunnableTaskCountID, "compute.googleapis.com/guest/cpu/runnable_task_count", "compute.googleapis.com"},
		{metrics.CPUUsageTimeID, "compute.googleapis.com/guest/cpu/usage_time", "compute.googleapis.com"},
		{metrics.CPULoad1m, "compute.googleapis.com/guest/cpu/load_1m", "compute.googleapis.com"},
		{metrics.CPULoad5m, "compute.googleapis.com/guest/cpu/load_5m", "compute.googleapis.com"},
		{metrics.CPULoad15m, "compute.googleapis.com/guest/cpu/load_15m", "compute.googleapis.com"},

		// Disk metrics
		{metrics.DiskAvgQueueLenID, "compute.googleapis.com/guest/disk/queue_length", "compute.googleapis.com"},
		{metrics.DiskBytesUsedID, "compute.googleapis.com/guest/disk/bytes_used", "compute.googleapis.com"},
		{metrics.DiskPercentUsedID, "compute.googleapis.com/guest/disk/percent_used", "compute.googleapis.com"},
		{metrics.DiskIOTimeID, "compute.googleapis.com/guest/disk/io_time", "compute.googleapis.com"},
		{metrics.DiskMergedOpsCountID, "compute.googleapis.com/guest/disk/merged_operation_count", "compute.googleapis.com"},
		{metrics.DiskOpsBytesID, "compute.googleapis.com/guest/disk/operation_bytes_count", "compute.googleapis.com"},
		{metrics.DiskOpsCountID, "compute.googleapis.com/guest/disk/operation_count", "compute.googleapis.com"},
		{metrics.DiskOpsTimeID, "compute.googleapis.com/guest/disk/operation_time", "compute.googleapis.com"},
		{metrics.DiskWeightedIOID, "compute.googleapis.com/guest/disk/weighted_io_time", "compute.googleapis.com"},

		// System metrics
		{metrics.HostUptimeID, "compute.googleapis.com/guest/system/uptime", "compute.googleapis.com"},
		{metrics.SystemProcessesTotal, "kubernetes.io/internal/node/guest/system/processes_total", "kubernetes.io"},
		{metrics.SystemProcsRunning, "kubernetes.io/internal/node/guest/system/procs_running", "kubernetes.io"},
		{metrics.SystemProcsBlocked, "kubernetes.io/internal/node/guest/system/procs_blocked", "kubernetes.io"},
		{metrics.SystemInterruptsTotal, "kubernetes.io/internal/node/guest/system/interrupts_total", "kubernetes.io"},
		{metrics.SystemCPUStat, "kubernetes.io/internal/node/guest/system/cpu_stat", "kubernetes.io"},

		// Memory metrics
		{metrics.MemoryAnonymousUsedID, "compute.googleapis.com/guest/memory/anonymous_used", "compute.googleapis.com"},
		{metrics.MemoryBytesUsedID, "compute.googleapis.com/guest/memory/bytes_used", "compute.googleapis.com"},
		{metrics.MemoryDirtyUsedID, "compute.googleapis.com/guest/memory/dirty_used", "compute.googleapis.com"},
		{metrics.MemoryPageCacheUsedID, "compute.googleapis.com/guest/memory/page_cache_used", "compute.googleapis.com"},
		{metrics.MemoryUnevictableUsedID, "compute.googleapis.com/guest/memory/unevictable_used", "compute.googleapis.com"},
		{metrics.MemoryPercentUsedID, "compute.googleapis.com/guest/memory/percent_used", "compute.googleapis.com"},

		// Problem metrics
		{metrics.ProblemCounterID, "compute.googleapis.com/guest/system/problem_count", "compute.googleapis.com"},
		{metrics.ProblemGaugeID, "compute.googleapis.com/guest/system/problem_state", "compute.googleapis.com"},
		{metrics.OSFeatureID, "compute.googleapis.com/guest/system/os_feature_enabled", "compute.googleapis.com"},

		// Network metrics
		{metrics.NetDevRxBytes, "kubernetes.io/internal/node/guest/net/rx_bytes", "kubernetes.io"},
		{metrics.NetDevRxPackets, "kubernetes.io/internal/node/guest/net/rx_packets", "kubernetes.io"},
		{metrics.NetDevRxErrors, "kubernetes.io/internal/node/guest/net/rx_errors", "kubernetes.io"},
		{metrics.NetDevRxDropped, "kubernetes.io/internal/node/guest/net/rx_dropped", "kubernetes.io"},
		{metrics.NetDevRxFifo, "kubernetes.io/internal/node/guest/net/rx_fifo", "kubernetes.io"},
		{metrics.NetDevRxFrame, "kubernetes.io/internal/node/guest/net/rx_frame", "kubernetes.io"},
		{metrics.NetDevRxCompressed, "kubernetes.io/internal/node/guest/net/rx_compressed", "kubernetes.io"},
		{metrics.NetDevRxMulticast, "kubernetes.io/internal/node/guest/net/rx_multicast", "kubernetes.io"},
		{metrics.NetDevTxBytes, "kubernetes.io/internal/node/guest/net/tx_bytes", "kubernetes.io"},
		{metrics.NetDevTxPackets, "kubernetes.io/internal/node/guest/net/tx_packets", "kubernetes.io"},
		{metrics.NetDevTxErrors, "kubernetes.io/internal/node/guest/net/tx_errors", "kubernetes.io"},
		{metrics.NetDevTxDropped, "kubernetes.io/internal/node/guest/net/tx_dropped", "kubernetes.io"},
		{metrics.NetDevTxFifo, "kubernetes.io/internal/node/guest/net/tx_fifo", "kubernetes.io"},
		{metrics.NetDevTxCollisions, "kubernetes.io/internal/node/guest/net/tx_collisions", "kubernetes.io"},
		{metrics.NetDevTxCarrier, "kubernetes.io/internal/node/guest/net/tx_carrier", "kubernetes.io"},
		{metrics.NetDevTxCompressed, "kubernetes.io/internal/node/guest/net/tx_compressed", "kubernetes.io"},
	}

	for _, tt := range tests {
		t.Run(string(tt.metricID), func(t *testing.T) {
			// Verify the mapping exists
			sdMetric, exists := NPDMetricToSDMetric[tt.metricID]
			require.True(t, exists, "metric ID %s not found in NPDMetricToSDMetric map", tt.metricID)

			// Verify the mapped value is correct
			assert.Equal(t, tt.expectedSDMetric, sdMetric, "incorrect Stackdriver metric type for %s", tt.metricID)

			// Verify the metric type uses expected prefix
			assert.Contains(t, sdMetric, tt.expectedPrefix, "metric type should contain prefix %s", tt.expectedPrefix)
		})
	}

	// Verify we tested all entries in the map
	assert.Equal(t, len(NPDMetricToSDMetric), len(tests), "test count should match NPDMetricToSDMetric map size")
}

// TestGetMetricTypeConversionFunction tests the metric name to Stackdriver metric type conversion.
func TestGetMetricTypeConversionFunction(t *testing.T) {
	// Set up metric mappings
	metrics.MetricMap.AddMapping(metrics.HostUptimeID, "host/uptime")
	metrics.MetricMap.AddMapping(metrics.CPULoad1m, "cpu/load_1m")
	metrics.MetricMap.AddMapping(metrics.MemoryBytesUsedID, "memory/bytes_used")
	metrics.MetricMap.AddMapping(metrics.DiskBytesUsedID, "disk/bytes_used")
	metrics.MetricMap.AddMapping(metrics.ProblemCounterID, "problem/counter")

	tests := []struct {
		name               string
		customMetricPrefix string
		metricViewName     string
		expectedMetricType string
	}{
		{
			name:               "known metric with no custom prefix",
			customMetricPrefix: "",
			metricViewName:     "host/uptime",
			expectedMetricType: "compute.googleapis.com/guest/system/uptime",
		},
		{
			name:               "known CPU metric",
			customMetricPrefix: "",
			metricViewName:     "cpu/load_1m",
			expectedMetricType: "compute.googleapis.com/guest/cpu/load_1m",
		},
		{
			name:               "known memory metric",
			customMetricPrefix: "",
			metricViewName:     "memory/bytes_used",
			expectedMetricType: "compute.googleapis.com/guest/memory/bytes_used",
		},
		{
			name:               "known disk metric",
			customMetricPrefix: "",
			metricViewName:     "disk/bytes_used",
			expectedMetricType: "compute.googleapis.com/guest/disk/bytes_used",
		},
		{
			name:               "known problem counter",
			customMetricPrefix: "",
			metricViewName:     "problem/counter",
			expectedMetricType: "compute.googleapis.com/guest/system/problem_count",
		},
		{
			name:               "unknown metric with no custom prefix returns empty",
			customMetricPrefix: "",
			metricViewName:     "unknown/metric",
			expectedMetricType: "",
		},
		{
			name:               "unknown metric with custom prefix",
			customMetricPrefix: "custom.googleapis.com/npd",
			metricViewName:     "unknown/metric",
			expectedMetricType: "custom.googleapis.com/npd/unknown/metric",
		},
		{
			name:               "known metric ignores custom prefix",
			customMetricPrefix: "custom.googleapis.com/npd",
			metricViewName:     "host/uptime",
			expectedMetricType: "compute.googleapis.com/guest/system/uptime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := getMetricTypeConversionFunction(tt.customMetricPrefix)
			result := converter(tt.metricViewName)
			assert.Equal(t, tt.expectedMetricType, result)
		})
	}
}

// TestMetricTypeFormatConsistency verifies that all Stackdriver metric types follow expected patterns.
func TestMetricTypeFormatConsistency(t *testing.T) {
	for metricID, sdMetric := range NPDMetricToSDMetric {
		t.Run(string(metricID), func(t *testing.T) {
			// All metric types should be non-empty
			assert.NotEmpty(t, sdMetric, "metric type should not be empty")

			// Should contain either compute.googleapis.com or kubernetes.io prefix
			hasValidPrefix := strings.Contains(sdMetric, "compute.googleapis.com") ||
				strings.Contains(sdMetric, "kubernetes.io")
			assert.True(t, hasValidPrefix, "metric type should have valid prefix: %s", sdMetric)

			// Should contain /guest/ in the path
			assert.Contains(t, sdMetric, "/guest/", "metric type should contain /guest/ path component")

			// Should not have trailing slash
			assert.False(t, strings.HasSuffix(sdMetric, "/"), "metric type should not end with /")

			// Should not have double slashes
			assert.NotContains(t, sdMetric, "//", "metric type should not contain //")
		})
	}
}

// TestCustomMetricPrefixFormatting verifies custom metric prefix is properly formatted.
func TestCustomMetricPrefixFormatting(t *testing.T) {
	tests := []struct {
		name               string
		customPrefix       string
		metricName         string
		expectedResult     string
	}{
		{
			name:           "simple custom prefix",
			customPrefix:   "custom.googleapis.com/npd",
			metricName:     "test/metric",
			expectedResult: "custom.googleapis.com/npd/test/metric",
		},
		{
			name:           "custom prefix with trailing slash",
			customPrefix:   "custom.googleapis.com/npd/",
			metricName:     "test/metric",
			expectedResult: "custom.googleapis.com/npd/test/metric",
		},
		{
			name:           "empty custom prefix",
			customPrefix:   "",
			metricName:     "test/metric",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := getMetricTypeConversionFunction(tt.customPrefix)
			result := converter(tt.metricName)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
