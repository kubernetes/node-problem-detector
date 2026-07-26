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
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

func TestDiskCollector(t *testing.T) {
	dc := NewDiskCollectorOrDie(&ssmtypes.DiskStatsConfig{})
	dc.collect()
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
	otelutil.AddMetricReader(sdkmetric.NewManualReader())
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
