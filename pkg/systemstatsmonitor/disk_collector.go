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

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type diskCollector struct {
	mIOTime         *metrics.Int64Metric
	mWeightedIO     *metrics.Int64Metric
	mAvgQueueLen    *metrics.Float64Metric
	mOpsCount       *metrics.Int64Metric
	mMergedOpsCount *metrics.Int64Metric
	mOpsBytes       *metrics.Int64Metric
	mOpsTime        *metrics.Int64Metric
	mBytesUsed      *metrics.Int64Metric

	config *ssmtypes.DiskStatsConfig

	lastIOTime           map[string]uint64
	lastWeightedIO       map[string]uint64
	lastReadCount        map[string]uint64
	lastWriteCount       map[string]uint64
	lastMergedReadCount  map[string]uint64
	lastMergedWriteCount map[string]uint64
	lastReadBytes        map[string]uint64
	lastWriteBytes       map[string]uint64
	lastReadTime         map[string]uint64
	lastWriteTime        map[string]uint64

	lastSampleTime time.Time
}

func NewDiskCollectorOrDie(diskConfig *ssmtypes.DiskStatsConfig) *diskCollector {
	dc := diskCollector{config: diskConfig}

	var err error

	// Use metrics.Sum aggregation method to ensure the metric is a counter/cumulative metric.
	dc.mIOTime, err = metrics.NewInt64Metric(
		metrics.DiskIOTimeID,
		diskConfig.MetricsConfigs[string(metrics.DiskIOTimeID)].DisplayName,
		"The IO time spent on the disk, in ms",
		"ms",
		metrics.Sum,
		[]string{deviceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for disk/io_time: %v", err)
	}

	// Use metrics.Sum aggregation method to ensure the metric is a counter/cumulative metric.
	dc.mWeightedIO, err = metrics.NewInt64Metric(
		metrics.DiskWeightedIOID,
		diskConfig.MetricsConfigs[string(metrics.DiskWeightedIOID)].DisplayName,
		"The weighted IO on the disk, in ms",
		"ms",
		metrics.Sum,
		[]string{deviceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for disk/weighted_io: %v", err)
	}

	dc.mAvgQueueLen, err = metrics.NewFloat64Metric(
		metrics.DiskAvgQueueLenID,
		diskConfig.MetricsConfigs[string(metrics.DiskAvgQueueLenID)].DisplayName,
		"The average queue length on the disk",
		"1",
		metrics.LastValue,
		[]string{deviceNameLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for disk/avg_queue_len: %v", err)
	}

	dc.mOpsCount, err = metrics.NewInt64Metric(
		metrics.DiskOpsCountID,
		diskConfig.MetricsConfigs[string(metrics.DiskOpsCountID)].DisplayName,
		"Disk operations count",
		"1",
		metrics.Sum,
		[]string{deviceNameLabel, directionLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.DiskOpsCountID, err)
	}

	dc.mMergedOpsCount, err = metrics.NewInt64Metric(
		metrics.DiskMergedOpsCountID,
		diskConfig.MetricsConfigs[string(metrics.DiskMergedOpsCountID)].DisplayName,
		"Disk merged operations count",
		"1",
		metrics.Sum,
		[]string{deviceNameLabel, directionLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.DiskMergedOpsCountID, err)
	}

	dc.mOpsBytes, err = metrics.NewInt64Metric(
		metrics.DiskOpsBytesID,
		diskConfig.MetricsConfigs[string(metrics.DiskOpsBytesID)].DisplayName,
		"Bytes transferred in disk operations",
		"1",
		metrics.Sum,
		[]string{deviceNameLabel, directionLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.DiskOpsBytesID, err)
	}

	dc.mOpsTime, err = metrics.NewInt64Metric(
		metrics.DiskOpsTimeID,
		diskConfig.MetricsConfigs[string(metrics.DiskOpsTimeID)].DisplayName,
		"Time spent in disk operations, in ms",
		"ms",
		metrics.Sum,
		[]string{deviceNameLabel, directionLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.DiskOpsTimeID, err)
	}

	dc.mBytesUsed, err = metrics.NewInt64Metric(
		metrics.DiskBytesUsedID,
		diskConfig.MetricsConfigs[string(metrics.DiskBytesUsedID)].DisplayName,
		"Disk bytes used, in Bytes",
		"Byte",
		metrics.LastValue,
		[]string{deviceNameLabel, fsTypeLabel, mountOptionLabel, stateLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.DiskBytesUsedID, err)
	}

	dc.lastIOTime = make(map[string]uint64)
	dc.lastWeightedIO = make(map[string]uint64)
	dc.lastReadCount = make(map[string]uint64)
	dc.lastWriteCount = make(map[string]uint64)
	dc.lastMergedReadCount = make(map[string]uint64)
	dc.lastMergedWriteCount = make(map[string]uint64)
	dc.lastReadBytes = make(map[string]uint64)
	dc.lastWriteBytes = make(map[string]uint64)
	dc.lastReadTime = make(map[string]uint64)
	dc.lastWriteTime = make(map[string]uint64)

	return &dc
}

func (dc *diskCollector) recordIOCounters(ioCountersStats map[string]disk.IOCountersStat, sampleTime time.Time) {
	for deviceName, ioCountersStat := range ioCountersStats {
		// Attach label {"device_name": deviceName} to the following metrics.
		tags := map[string]string{deviceNameLabel: deviceName}

		// Calculate average IO queue length since last measurement.
		lastIOTime, historyExist := dc.lastIOTime[deviceName]
		lastWeightedIO := dc.lastWeightedIO[deviceName]
		dc.lastIOTime[deviceName] = ioCountersStat.IoTime
		dc.lastWeightedIO[deviceName] = ioCountersStat.WeightedIO

		if dc.mIOTime != nil {
			dc.mIOTime.Record(tags, int64(ioCountersStat.IoTime-lastIOTime))
		}
		if dc.mWeightedIO != nil {
			dc.mWeightedIO.Record(tags, int64(ioCountersStat.WeightedIO-lastWeightedIO))
		}
		if historyExist {
			avgQueueLen := float64(0.0)
			if lastWeightedIO != ioCountersStat.WeightedIO {
				diffSampleTimeMs := sampleTime.Sub(dc.lastSampleTime).Seconds() * 1000
				avgQueueLen = float64(ioCountersStat.WeightedIO-lastWeightedIO) / diffSampleTimeMs
			}
			if dc.mAvgQueueLen != nil {
				dc.mAvgQueueLen.Record(tags, avgQueueLen)
			}
		}

		// Attach label {"device_name": deviceName, "direction": "read"} to the following metrics.
		tags = map[string]string{deviceNameLabel: deviceName, directionLabel: "read"}

		if dc.mOpsCount != nil {
			dc.mOpsCount.Record(tags, int64(ioCountersStat.ReadCount-dc.lastReadCount[deviceName]))
			dc.lastReadCount[deviceName] = ioCountersStat.ReadCount
		}
		if dc.mMergedOpsCount != nil {
			dc.mMergedOpsCount.Record(tags, int64(ioCountersStat.MergedReadCount-dc.lastMergedReadCount[deviceName]))
			dc.lastMergedReadCount[deviceName] = ioCountersStat.MergedReadCount
		}
		if dc.mOpsBytes != nil {
			dc.mOpsBytes.Record(tags, int64(ioCountersStat.ReadBytes-dc.lastReadBytes[deviceName]))
			dc.lastReadBytes[deviceName] = ioCountersStat.ReadBytes
		}
		if dc.mOpsTime != nil {
			dc.mOpsTime.Record(tags, int64(ioCountersStat.ReadTime-dc.lastReadTime[deviceName]))
			dc.lastReadTime[deviceName] = ioCountersStat.ReadTime
		}

		// Attach label {"device_name": deviceName, "direction": "write"} to the following metrics.
		tags = map[string]string{deviceNameLabel: deviceName, directionLabel: "write"}

		if dc.mOpsCount != nil {
			dc.mOpsCount.Record(tags, int64(ioCountersStat.WriteCount-dc.lastWriteCount[deviceName]))
			dc.lastWriteCount[deviceName] = ioCountersStat.WriteCount
		}
		if dc.mMergedOpsCount != nil {
			dc.mMergedOpsCount.Record(tags, int64(ioCountersStat.MergedWriteCount-dc.lastMergedWriteCount[deviceName]))
			dc.lastMergedWriteCount[deviceName] = ioCountersStat.MergedWriteCount
		}
		if dc.mOpsBytes != nil {
			dc.mOpsBytes.Record(tags, int64(ioCountersStat.WriteBytes-dc.lastWriteBytes[deviceName]))
			dc.lastWriteBytes[deviceName] = ioCountersStat.WriteBytes
		}
		if dc.mOpsTime != nil {
			dc.mOpsTime.Record(tags, int64(ioCountersStat.WriteTime-dc.lastWriteTime[deviceName]))
			dc.lastWriteTime[deviceName] = ioCountersStat.WriteTime
		}
	}
}

func (dc *diskCollector) collect() {
	if dc == nil {
		return
	}

	// List available devices.
	devices := []string{}
	if dc.config.IncludeRootBlk {
		devices = append(devices, listRootBlockDevices(dc.config.LsblkTimeout)...)
	}
	if dc.config.IncludeAllAttachedBlk {
		devices = append(devices, listAttachedBlockDevices()...)
	}

	// Fetch metrics from /proc, /sys.
	ioCountersStats, err := disk.IOCounters(devices...)
	if err != nil {
		glog.Errorf("Failed to retrieve disk IO counters: %v", err)
		return
	}
	partitions, err := disk.Partitions(false)
	if err != nil {
		glog.Errorf("Failed to list disk partitions: %v", err)
		return
	}
	sampleTime := time.Now()
	defer func() { dc.lastSampleTime = sampleTime }()

	// Record metrics regarding disk IO.
	dc.recordIOCounters(ioCountersStats, sampleTime)

	// Record metrics regarding disk space usage.
	if dc.mBytesUsed == nil {
		return
	}

	// to make sure that the rows are not duplicated
	// we display only the only one row even if there are
	// mutiple rows for the same disk.
	seen := make(map[string]bool)
	for _, partition := range partitions {
		if seen[partition.Device] {
			continue
		}
		seen[partition.Device] = true
		usageStat, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			glog.Errorf("Failed to retrieve disk usage for %q: %v", partition.Mountpoint, err)
			continue
		}
		deviceName := strings.TrimPrefix(partition.Device, "/dev/")
		fstype := partition.Fstype
		opttypes := partition.Opts
		dc.mBytesUsed.Record(map[string]string{deviceNameLabel: deviceName, fsTypeLabel: fstype, mountOptionLabel: opttypes, stateLabel: "free"}, int64(usageStat.Free))
		dc.mBytesUsed.Record(map[string]string{deviceNameLabel: deviceName, fsTypeLabel: fstype, mountOptionLabel: opttypes, stateLabel: "used"}, int64(usageStat.Used))
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
