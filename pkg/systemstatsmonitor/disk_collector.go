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

	var err error
	dc.keyDevice, err = tag.NewKey("device")
	if err != nil {
		glog.Fatalf("Failed to create device tag during initializing disk collector: %v", err)
	}

	if diskConfig.MetricsConfigs["disk/io_time"].DisplayName != "" {
		dc.mIOTime = metrics.NewInt64Metric(
			diskConfig.MetricsConfigs["disk/io_time"].DisplayName,
			"The IO time spent on the disk",
			"second",
			view.LastValue(),
			[]tag.Key{dc.keyDevice})
	}

	if diskConfig.MetricsConfigs["disk/weighted_io"].DisplayName != "" {
		dc.mWeightedIO = metrics.NewInt64Metric(
			diskConfig.MetricsConfigs["disk/weighted_io"].DisplayName,
			"The weighted IO on the disk",
			"second",
			view.LastValue(),
			[]tag.Key{dc.keyDevice})
	}

	if diskConfig.MetricsConfigs["disk/avg_queue_len"].DisplayName != "" {
		dc.mAvgQueueLen = metrics.NewFloat64Metric(
			diskConfig.MetricsConfigs["disk/avg_queue_len"].DisplayName,
			"The average queue length on the disk",
			"second",
			view.LastValue(),
			[]tag.Key{dc.keyDevice})
	}

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

	ioCountersStats, err := disk.IOCounters(blks...)
	if err != nil {
		glog.Errorf("Failed to retrieve disk IO counters: %v", err)
		return
	}

	for deviceName, ioCountersStat := range ioCountersStats {
		// Calculate average IO queue length since last measurement.
		lastIOTime := dc.historyIOTime[deviceName]
		lastWeightedIO := dc.historyWeightedIO[deviceName]

		dc.historyIOTime[deviceName] = ioCountersStat.IoTime
		dc.historyWeightedIO[deviceName] = ioCountersStat.WeightedIO

		avgQueueLen := float64(0.0)
		if lastIOTime != ioCountersStat.IoTime {
			avgQueueLen = float64(ioCountersStat.WeightedIO-lastWeightedIO) / float64(ioCountersStat.IoTime-lastIOTime)
		}

		// Attach label {"device": deviceName} to the metrics.
		deviceCtx, err := tag.New(context.Background(), tag.Upsert(dc.keyDevice, deviceName))
		if err != nil {
			glog.Errorf("Failed to create context with device tag: %v", err)
			deviceCtx = context.Background()
		}

		if dc.mIOTime != nil {
			stats.Record(deviceCtx, dc.mIOTime.M(int64(ioCountersStat.IoTime)))
		}
		if dc.mWeightedIO != nil {
			stats.Record(deviceCtx, dc.mWeightedIO.M(int64(ioCountersStat.WeightedIO)))
		}
		if dc.mAvgQueueLen != nil {
			stats.Record(deviceCtx, dc.mAvgQueueLen.M(avgQueueLen))
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
	blks := []string{}

	partitions, err := disk.Partitions(false)
	if err != nil {
		glog.Errorf("Failed to retrieve the list of disk partitions: %v", err)
		return blks
	}

	for _, partition := range partitions {
		blks = append(blks, partition.Device)
	}
	return blks
}
