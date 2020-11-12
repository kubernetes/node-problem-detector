/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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
	"github.com/golang/glog"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

// clockTick is the ratio between 1 second and 1 USER_HZ (a clock tick).
//
// CLK_TCK is 100 in most architectures. If NPD ever runs on a super special architecture,
// we can work out a way to detect the clock tick on that architecture (might require
// cross-compilation with C library or parsing kernel ABIs). For now, it's not worth the
// complexity.
//
// See documentation at http://man7.org/linux/man-pages/man5/proc.5.html
const clockTick float64 = 100.0

type cpuCollector struct {
	mRunnableTaskCount *metrics.Float64Metric
	mUsageTime         *metrics.Float64Metric
	mCpuLoad1m         *metrics.Float64Metric
	mCpuLoad5m         *metrics.Float64Metric
	mCpuLoad15m        *metrics.Float64Metric

	config *ssmtypes.CPUStatsConfig

	lastUsageTime map[string]float64
}

func NewCPUCollectorOrDie(cpuConfig *ssmtypes.CPUStatsConfig) *cpuCollector {
	cc := cpuCollector{config: cpuConfig}

	var err error

	cc.mRunnableTaskCount, err = metrics.NewFloat64Metric(
		metrics.CPURunnableTaskCountID,
		cpuConfig.MetricsConfigs[string(metrics.CPURunnableTaskCountID)].DisplayName,
		"The average number of runnable tasks in the run-queue during the last minute",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CPURunnableTaskCountID, err)
	}

	cc.mUsageTime, err = metrics.NewFloat64Metric(
		metrics.CPUUsageTimeID,
		cpuConfig.MetricsConfigs[string(metrics.CPUUsageTimeID)].DisplayName,
		"CPU usage, in seconds",
		"s",
		metrics.Sum,
		[]string{stateLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CPUUsageTimeID, err)
	}

	cc.mCpuLoad1m, err = metrics.NewFloat64Metric(
		metrics.CPULoad1m,
		cpuConfig.MetricsConfigs[string(metrics.CPULoad1m)].DisplayName,
		"CPU average load (1m)",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CPULoad1m, err)
	}

	cc.mCpuLoad5m, err = metrics.NewFloat64Metric(
		metrics.CPULoad5m,
		cpuConfig.MetricsConfigs[string(metrics.CPULoad5m)].DisplayName,
		"CPU average load (5m)",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CPULoad5m, err)
	}

	cc.mCpuLoad15m, err = metrics.NewFloat64Metric(
		metrics.CPULoad15m,
		cpuConfig.MetricsConfigs[string(metrics.CPULoad15m)].DisplayName,
		"CPU average load (15m)",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.CPULoad15m, err)
	}

	cc.lastUsageTime = make(map[string]float64)

	return &cc
}

func (cc *cpuCollector) recordLoad() {
	if cc.mRunnableTaskCount == nil {
		return
	}

	loadAvg, err := load.Avg()
	if err != nil {
		glog.Errorf("Failed to retrieve average CPU load: %v", err)
		return
	}

	cc.mRunnableTaskCount.Record(map[string]string{}, loadAvg.Load1)

	cc.mCpuLoad1m.Record(map[string]string{}, loadAvg.Load1)
	cc.mCpuLoad5m.Record(map[string]string{}, loadAvg.Load5)
	cc.mCpuLoad15m.Record(map[string]string{}, loadAvg.Load15)
}

func (cc *cpuCollector) recordUsage() {
	if cc.mUsageTime == nil {
		return
	}

	// Set percpu=false to get aggregated usage from all CPUs.
	timersStats, err := cpu.Times(false)
	if err != nil {
		glog.Errorf("Failed to retrieve CPU timers stat: %v", err)
		return
	}
	timersStat := timersStats[0]

	cc.mUsageTime.Record(map[string]string{stateLabel: "user"}, clockTick*timersStat.User-cc.lastUsageTime["user"])
	cc.lastUsageTime["user"] = clockTick * timersStat.User

	cc.mUsageTime.Record(map[string]string{stateLabel: "system"}, clockTick*timersStat.System-cc.lastUsageTime["system"])
	cc.lastUsageTime["system"] = clockTick * timersStat.System

	cc.mUsageTime.Record(map[string]string{stateLabel: "idle"}, clockTick*timersStat.Idle-cc.lastUsageTime["idle"])
	cc.lastUsageTime["idle"] = clockTick * timersStat.Idle

	cc.mUsageTime.Record(map[string]string{stateLabel: "nice"}, clockTick*timersStat.Nice-cc.lastUsageTime["nice"])
	cc.lastUsageTime["nice"] = clockTick * timersStat.Nice

	cc.mUsageTime.Record(map[string]string{stateLabel: "iowait"}, clockTick*timersStat.Iowait-cc.lastUsageTime["iowait"])
	cc.lastUsageTime["iowait"] = clockTick * timersStat.Iowait

	cc.mUsageTime.Record(map[string]string{stateLabel: "irq"}, clockTick*timersStat.Irq-cc.lastUsageTime["irq"])
	cc.lastUsageTime["irq"] = clockTick * timersStat.Irq

	cc.mUsageTime.Record(map[string]string{stateLabel: "softirq"}, clockTick*timersStat.Softirq-cc.lastUsageTime["softirq"])
	cc.lastUsageTime["softirq"] = clockTick * timersStat.Softirq

	cc.mUsageTime.Record(map[string]string{stateLabel: "steal"}, clockTick*timersStat.Steal-cc.lastUsageTime["steal"])
	cc.lastUsageTime["steal"] = clockTick * timersStat.Steal

	cc.mUsageTime.Record(map[string]string{stateLabel: "guest"}, clockTick*timersStat.Guest-cc.lastUsageTime["guest"])
	cc.lastUsageTime["guest"] = clockTick * timersStat.Guest

	cc.mUsageTime.Record(map[string]string{stateLabel: "guest_nice"}, clockTick*timersStat.GuestNice-cc.lastUsageTime["guest_nice"])
	cc.lastUsageTime["guest_nice"] = clockTick * timersStat.GuestNice
}

func (cc *cpuCollector) collect() {
	if cc == nil {
		return
	}

	cc.recordLoad()
	cc.recordUsage()
}
