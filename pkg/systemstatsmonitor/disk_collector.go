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

package systemstatsmonitor

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/shirou/gopsutil/disk"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type diskCollector struct {
	keyDevice    tag.Key
	mIOTime      *stats.Int64Measure
	mWeightedIO  *stats.Int64Measure
	mAvgQueueLen *stats.Float64Measure

	config *ssmtypes.DiskStatsConfig

	historyIOTime     map[string]uint64
	historyWeightedIO map[string]uint64
}

func NewDiskCollectorOrDie(diskConfig *ssmtypes.DiskStatsConfig) *diskCollector {
	dc := diskCollector{config: diskConfig}
	dc.keyDevice, _ = tag.NewKey("device")

	dc.mIOTime = metrics.NewInt64Metric(
		diskConfig.MetricsConfigs["disk/io_time"].DisplayName,
		"The IO time spent on the disk",
		"second",
		view.LastValue(),
		[]tag.Key{dc.keyDevice})

	dc.mWeightedIO = metrics.NewInt64Metric(
		diskConfig.MetricsConfigs["disk/weighted_io"].DisplayName,
		"The weighted IO on the disk",
		"second",
		view.LastValue(),
		[]tag.Key{dc.keyDevice})

	dc.mAvgQueueLen = metrics.NewFloat64Metric(
		diskConfig.MetricsConfigs["disk/avg_queue_len"].DisplayName,
		"The average queue length on the disk",
		"second",
		view.LastValue(),
		[]tag.Key{dc.keyDevice})

	dc.historyIOTime = make(map[string]uint64)
	dc.historyWeightedIO = make(map[string]uint64)

	return &dc
}

func (dc *diskCollector) collect() {
	if dc == nil {
		return
	}

	blks := []string{}
	if dc.config.IncludeRootBlk {
		blks = append(blks, listRootBlockDevices(dc.config.LsblkTimeout)...)
	}
	if dc.config.IncludeAllAttachedBlk {
		blks = append(blks, listAttachedBlockDevices()...)
	}

	ioCountersStats, _ := disk.IOCounters(blks...)

	for deviceName, ioCountersStat := range ioCountersStats {
		// Calculate average IO queue length since last measurement.
		lastIOTime := dc.historyIOTime[deviceName]
		lastWeightedIO := dc.historyWeightedIO[deviceName]

		dc.historyIOTime[deviceName] = ioCountersStat.IoTime
		dc.historyWeightedIO[deviceName] = ioCountersStat.WeightedIO

		avg_queue_len := float64(0.0)
		if lastIOTime != ioCountersStat.IoTime {
			avg_queue_len = float64(ioCountersStat.WeightedIO-lastWeightedIO) / float64(ioCountersStat.IoTime-lastIOTime)
		}

		// Attach label {"device": deviceName} to the metrics.
		device_ctx, _ := tag.New(context.Background(), tag.Upsert(dc.keyDevice, deviceName))
		if dc.mIOTime != nil {
			stats.Record(device_ctx, dc.mIOTime.M(int64(ioCountersStat.IoTime)))
		}
		if dc.mWeightedIO != nil {
			stats.Record(device_ctx, dc.mWeightedIO.M(int64(ioCountersStat.WeightedIO)))
		}
		if dc.mAvgQueueLen != nil {
			stats.Record(device_ctx, dc.mAvgQueueLen.M(avg_queue_len))
		}
	}
}

// listRootBlockDevices lists all block devices that's not a slave or holder.
func listRootBlockDevices(timeout time.Duration) []string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// "-d" prevents printing slave or holder devices. i.e. /dev/sda1, /dev/sda2...
	// "-n" prevents printing the headings.
	// "-p NAME" specifies to only print the device name.
	cmd := exec.CommandContext(ctx, "lsblk", "-d", "-n", "-o", "NAME")
	stdout, err := cmd.Output()
	if err != nil {
		glog.Errorf("Error calling lsblk")
	}
	return strings.Split(strings.TrimSpace(string(stdout)), "\n")
}

// listAttachedBlockDevices lists all currently attached block devices.
func listAttachedBlockDevices() []string {
	partitions, _ := disk.Partitions(false)
	blks := []string{}
	for _, partition := range partitions {
		blks = append(blks, partition.Device)
	}
	return blks
}
