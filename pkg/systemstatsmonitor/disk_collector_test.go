/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package systemstatsmonitor

import (
	"context"
	"testing"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

func TestDiskCollector(t *testing.T) {
	dc := NewDiskCollectorOrDie(&ssmtypes.DiskStatsConfig{})
	dc.collect()
}

func TestDiskCollectorUsageMetrics(t *testing.T) {
	for _, tc := range []struct {
		name    string
		bytes   bool
		percent bool
	}{
		{name: "bytes only", bytes: true},
		{name: "percent only", percent: true},
		{name: "both", bytes: true, percent: true},
		{name: "neither"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			otelutil.ResetForTesting()
			defer otelutil.ResetForTesting()
			reader := sdkmetric.NewManualReader()
			require.NoError(t, otelutil.AddMetricReader(reader))
			otelutil.InitializeMeterProvider()

			config := &ssmtypes.DiskStatsConfig{
				MetricsConfigs: map[string]ssmtypes.MetricConfig{},
			}
			if tc.bytes {
				config.MetricsConfigs[string(metrics.DiskBytesUsedID)] = ssmtypes.MetricConfig{DisplayName: string(metrics.DiskBytesUsedID)}
			}
			if tc.percent {
				config.MetricsConfigs[string(metrics.DiskPercentUsedID)] = ssmtypes.MetricConfig{DisplayName: string(metrics.DiskPercentUsedID)}
			}
			dc := NewDiskCollectorOrDie(config)
			dc.recordDiskUsage([]disk.PartitionStat{{
				Device: "/dev/testdisk", Mountpoint: t.TempDir(), Fstype: "testfs", Opts: []string{"rw"},
			}})

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(context.Background(), &rm))
			foundBytes, foundPercent := false, false
			for _, scope := range rm.ScopeMetrics {
				for _, metric := range scope.Metrics {
					switch metric.Name {
					case "disk_bytes_used":
						foundBytes = true
						gauge, ok := metric.Data.(metricdata.Gauge[int64])
						require.True(t, ok)
						require.Len(t, gauge.DataPoints, 2)
						states := []string{}
						for _, point := range gauge.DataPoints {
							state, ok := point.Attributes.Value(attribute.Key(stateLabel))
							require.True(t, ok)
							states = append(states, state.AsString())
							assert.Equal(t, attribute.NewSet(
								attribute.String(deviceNameLabel, "testdisk"),
								attribute.String(fsTypeLabel, "testfs"),
								attribute.String(mountOptionLabel, "rw"),
								attribute.String(stateLabel, state.AsString()),
							), point.Attributes)
							assert.GreaterOrEqual(t, point.Value, int64(0))
						}
						assert.ElementsMatch(t, []string{"free", "used"}, states)
					case "disk_percent_used":
						foundPercent = true
						gauge, ok := metric.Data.(metricdata.Gauge[float64])
						require.True(t, ok)
						require.Len(t, gauge.DataPoints, 1)
						point := gauge.DataPoints[0]
						assert.Equal(t, attribute.NewSet(attribute.String(deviceNameLabel, "testdisk")), point.Attributes)
						assert.GreaterOrEqual(t, point.Value, 0.0)
						assert.LessOrEqual(t, point.Value, 100.0)
					}
				}
			}
			assert.Equal(t, tc.bytes, foundBytes, "disk_bytes_used presence")
			assert.Equal(t, tc.percent, foundPercent, "disk_percent_used presence")
		})
	}
}

// TestDiskCollectorPercentUsedLabels pins the label schema of disk_percent_used
// to device_name only. NPD has exported it that way since the metric was
// introduced (the OpenCensus view aggregated by device_name), and it maps to the
// Google-owned compute.googleapis.com/guest/disk/percent_used descriptor, which
// can reject writes carrying undeclared label keys. Because Record now rejects
// undeclared labels, widening the declared set is the only way collect() could
// emit a wider schema — so the metric must accept device_name and reject more.
func TestDiskCollectorPercentUsedLabels(t *testing.T) {
	otelutil.ResetForTesting()
	defer otelutil.ResetForTesting()
	if err := otelutil.AddMetricReader(sdkmetric.NewManualReader()); err != nil {
		t.Fatalf("Failed to add metric reader: %v", err)
	}
	otelutil.InitializeMeterProvider()

	dc := NewDiskCollectorOrDie(&ssmtypes.DiskStatsConfig{
		MetricsConfigs: map[string]ssmtypes.MetricConfig{
			string(metrics.DiskPercentUsedID): {DisplayName: string(metrics.DiskPercentUsedID)},
		},
	})

	// collect() records disk_percent_used keyed only by device_name.
	if err := dc.mPercentUsed.Record(map[string]string{deviceNameLabel: "sda1"}, 42.0); err != nil {
		t.Fatalf("recording disk_percent_used with device_name failed: %v", err)
	}

	// A wider label set must be rejected, keeping the exported schema stable.
	if err := dc.mPercentUsed.Record(map[string]string{deviceNameLabel: "sda1", fsTypeLabel: "ext4"}, 42.0); err == nil {
		t.Error("disk_percent_used accepted an fs_type label; its schema must stay device_name-only")
	}
}
