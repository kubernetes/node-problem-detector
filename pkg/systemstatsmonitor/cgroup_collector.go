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
	"fmt"

	"github.com/golang/glog"
	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/cgroup"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type cgroupCollector struct {
	tags map[string]string

	mCgroupCPUPeriodsCount          *metrics.Int64Metric
	mCgroupCPUThrottledPeriodsCount *metrics.Int64Metric
	mCgroupCPUThrottledTime         *metrics.Int64Metric
	mCgroupCPUUsagePerCPU           *metrics.Int64Metric
	mCgroupCPUTime                  *metrics.Int64Metric
	mCgroupMemoryUsage              *metrics.Int64Metric
	mCgroupMemoryLimit              *metrics.Int64Metric
	mCgroupMemoryFailcntCount       *metrics.Int64Metric
	mCgroupPidsCurrentCount         *metrics.Int64Metric

	config *ssmtypes.CgroupStatsConfig
}

func NewCgroupCollectorOrDie(config *ssmtypes.CgroupStatsConfig) *cgroupCollector {
	cc := cgroupCollector{tags: map[string]string{}, config: config}

	var err error

	cc.mCgroupCPUPeriodsCount, err = metrics.NewInt64Metric(
		metrics.CgroupCPUPeriodsCount,
		config.MetricsConfigs[string(metrics.CgroupCPUPeriodsCount)].DisplayName,
		"Cumulative number of periods that have elapsed.",
		"1",
		metrics.Sum,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupCPUPeriodsCount, err)
	}

	cc.mCgroupCPUThrottledPeriodsCount, err = metrics.NewInt64Metric(
		metrics.CgroupCPUThrottledPeriodsCount,
		config.MetricsConfigs[string(metrics.CgroupCPUThrottledPeriodsCount)].DisplayName,
		"Cumulative number of periods where groups were throttled.",
		"1",
		metrics.Sum,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupCPUThrottledPeriodsCount, err)
	}

	cc.mCgroupCPUThrottledTime, err = metrics.NewInt64Metric(
		metrics.CgroupCPUThrottledTime,
		config.MetricsConfigs[string(metrics.CgroupCPUThrottledTime)].DisplayName,
		"Cumulative time (in nanoseconds) the groups were throttled.",
		"ns",
		metrics.Sum,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupCPUThrottledTime, err)
	}

	cc.mCgroupCPUUsagePerCPU, err = metrics.NewInt64Metric(
		metrics.CgroupCPUUsagePerCPU,
		config.MetricsConfigs[string(metrics.CgroupCPUUsagePerCPU)].DisplayName,
		"The cumulative CPU time, in nanoseconds, consumed by all tasks in this group, separated by CPU.",
		"ns",
		metrics.Sum,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel, cpuLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupCPUUsagePerCPU, err)
	}

	cc.mCgroupCPUTime, err = metrics.NewInt64Metric(
		metrics.CgroupCPUTime,
		config.MetricsConfigs[string(metrics.CgroupCPUTime)].DisplayName,
		"The cumulative user and system time consumed by tasks in this group.",
		"1",
		metrics.Sum,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel, systemLabel, userLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupCPUTime, err)
	}

	cc.mCgroupMemoryUsage, err = metrics.NewInt64Metric(
		metrics.CgroupMemoryUsage,
		config.MetricsConfigs[string(metrics.CgroupMemoryUsage)].DisplayName,
		"Instantaneous memory in bytes used by this cgroup.",
		"bytes",
		metrics.LastValue,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel, typeLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupMemoryUsage, err)
	}

	cc.mCgroupMemoryLimit, err = metrics.NewInt64Metric(
		metrics.CgroupMemoryLimit,
		config.MetricsConfigs[string(metrics.CgroupMemoryLimit)].DisplayName,
		"The memory limit in bytes of this group.",
		"bytes",
		metrics.LastValue,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupMemoryLimit, err)
	}

	cc.mCgroupMemoryFailcntCount, err = metrics.NewInt64Metric(
		metrics.CgroupMemoryFailcntCount,
		config.MetricsConfigs[string(metrics.CgroupMemoryFailcntCount)].DisplayName,
		"The number of memory usage hits limits.",
		"1",
		metrics.LastValue,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupMemoryFailcntCount, err)
	}

	cc.mCgroupPidsCurrentCount, err = metrics.NewInt64Metric(
		metrics.CgroupPidsCurrentCount,
		config.MetricsConfigs[string(metrics.CgroupPidsCurrentCount)].DisplayName,
		"The number of processes currently in the cgroup.",
		"1",
		metrics.LastValue,
		[]string{podIDLabel, containerIDLabel, serviceNameLabel},
	)
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CgroupPidsCurrentCount, err)
	}

	return &cc
}

func (cc *cgroupCollector) collect() {
	if cc == nil {
		return
	}

	allCgroups, err := cgroup.AllKubeCgroups()
	if err != nil {
		glog.Errorf("Failed to get kube cgroup paths: %v", err)
		return
	}

	for _, p := range allCgroups {
		cc.tags[podIDLabel] = p.PodID
		cc.tags[containerIDLabel] = p.ContainerID

		control, err := p.CgroupStats()
		if err != nil {
			glog.Errorf("Failed to get stats for cgroup %q: %v", p.CgroupPath(), err)
			return
		}

		cc.mCgroupCPUPeriodsCount.Record(cc.tags, int64(control.CPU.Throttling.Periods))
		cc.mCgroupCPUThrottledPeriodsCount.Record(cc.tags, int64(control.CPU.Throttling.ThrottledPeriods))
		cc.mCgroupCPUThrottledTime.Record(cc.tags, int64(control.CPU.Throttling.ThrottledTime))

		for i, pc := range control.CPU.Usage.PerCPU {
			tags := cc.tags
			tags[cpuLabel] = fmt.Sprintf("cpu%d", i)
			cc.mCgroupCPUUsagePerCPU.Record(tags, int64(pc))
		}

		tags := cc.tags
		tags[userLabel] = "true"
		tags[systemLabel] = "false"
		cc.mCgroupCPUTime.Record(tags, int64(control.CPU.Usage.User))
		tags[userLabel] = "false"
		tags[systemLabel] = "true"
		cc.mCgroupCPUTime.Record(tags, int64(control.CPU.Usage.Kernel))

		tags = cc.tags
		cc.mCgroupMemoryUsage.Record(tags, int64(control.Memory.Usage.Usage))
		tags[typeLabel] = "rss"
		cc.mCgroupMemoryUsage.Record(tags, int64(control.Memory.RSS))
		tags[typeLabel] = "cache"
		cc.mCgroupMemoryUsage.Record(tags, int64(control.Memory.Cache))
		tags[typeLabel] = "swap"
		cc.mCgroupMemoryUsage.Record(tags, int64(control.Memory.Swap.Usage))

		cc.mCgroupMemoryLimit.Record(cc.tags, int64(control.Memory.Usage.Limit))
		cc.mCgroupMemoryFailcntCount.Record(cc.tags, int64(control.Memory.Usage.Failcnt))

		cc.mCgroupPidsCurrentCount.Record(cc.tags, int64(control.Pids.Current))
	}
}
