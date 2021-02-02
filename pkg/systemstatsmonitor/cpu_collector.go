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
	"fmt"

	"github.com/golang/glog"
	"github.com/prometheus/procfs"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
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
	tags map[string]string

	mRunnableTaskCount     *metrics.Float64Metric
	mUsageTime             *metrics.Float64Metric
	mCpuLoad1m             *metrics.Float64Metric
	mCpuLoad5m             *metrics.Float64Metric
	mCpuLoad15m            *metrics.Float64Metric
	mSystemProcessesTotal  *metrics.Int64Metric
	mSystemProcsRunning    *metrics.Int64Metric
	mSystemProcsBlocked    *metrics.Int64Metric
	mSystemInterruptsTotal *metrics.Int64Metric
	mSystemCPUStat         *metrics.Float64Metric // per-cpu time from /proc/stats

	config *ssmtypes.CPUStatsConfig

	lastUsageTime map[string]float64
}

func NewCPUCollectorOrDie(cpuConfig *ssmtypes.CPUStatsConfig) *cpuCollector {
	cc := cpuCollector{tags: map[string]string{}, config: cpuConfig}

	kernelVersion, err := host.KernelVersion()
	if err != nil {
		glog.Fatalf("Failed to retrieve kernel version: %v", err)
	}
	cc.tags[kernelVersionLabel] = kernelVersion

	osVersion, err := util.GetOSVersion()
	if err != nil {
		glog.Fatalf("Failed to retrieve OS version: %v", err)
	}
	cc.tags[osVersionLabel] = osVersion

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

	cc.mSystemProcessesTotal, err = metrics.NewInt64Metric(
		metrics.SystemProcessesTotal,
		cpuConfig.MetricsConfigs[string(metrics.SystemProcessesTotal)].DisplayName,
		"Number of forks since boot.",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.SystemProcessesTotal, err)
	}

	cc.mSystemProcsRunning, err = metrics.NewInt64Metric(
		metrics.SystemProcsRunning,
		cpuConfig.MetricsConfigs[string(metrics.SystemProcsRunning)].DisplayName,
		"Number of processes currently running.",
		"1",
		metrics.LastValue,
		[]string{osVersionLabel, kernelVersionLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.SystemProcsRunning, err)
	}

	cc.mSystemProcsBlocked, err = metrics.NewInt64Metric(
		metrics.SystemProcsBlocked,
		cpuConfig.MetricsConfigs[string(metrics.SystemProcsBlocked)].DisplayName,
		"Number of processes currently blocked.",
		"1",
		metrics.LastValue,
		[]string{osVersionLabel, kernelVersionLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.SystemProcsBlocked, err)
	}

	cc.mSystemInterruptsTotal, err = metrics.NewInt64Metric(
		metrics.SystemInterruptsTotal,
		cpuConfig.MetricsConfigs[string(metrics.SystemInterruptsTotal)].DisplayName,
		"Total number of interrupts serviced (cumulative).",
		"1",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.SystemInterruptsTotal, err)
	}

	cc.mSystemCPUStat, err = metrics.NewFloat64Metric(
		metrics.SystemCPUStat,
		cpuConfig.MetricsConfigs[string(metrics.SystemCPUStat)].DisplayName,
		"Cumulative time each cpu spent in various stages.",
		"ns",
		metrics.Sum,
		[]string{osVersionLabel, kernelVersionLabel, cpuLabel, stageLabel})
	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.SystemCPUStat, err)
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

func (cc *cpuCollector) recordSystemStats() {
	fs, err := procfs.NewFS("/proc")
	stats, err := fs.Stat()
	if err != nil {
		glog.Errorf("Failed to retrieve cpu/process stats: %v", err)
		return
	}

	cc.mSystemProcessesTotal.Record(cc.tags, int64(stats.ProcessCreated))
	cc.mSystemProcsRunning.Record(cc.tags, int64(stats.ProcessesRunning))
	cc.mSystemProcsBlocked.Record(cc.tags, int64(stats.ProcessesBlocked))
	cc.mSystemInterruptsTotal.Record(cc.tags, int64(stats.IRQTotal))

	for i, c := range stats.CPU {
		tags := cc.tags
		tags[cpuLabel] = fmt.Sprintf("cpu%d", i)

		tags[stageLabel] = "user"
		cc.mSystemCPUStat.Record(tags, c.User)
		tags[stageLabel] = "nice"
		cc.mSystemCPUStat.Record(tags, c.Nice)
		tags[stageLabel] = "system"
		cc.mSystemCPUStat.Record(tags, c.System)
		tags[stageLabel] = "idle"
		cc.mSystemCPUStat.Record(tags, c.Idle)
		tags[stageLabel] = "iowait"
		cc.mSystemCPUStat.Record(tags, c.Iowait)
		tags[stageLabel] = "iRQ"
		cc.mSystemCPUStat.Record(tags, c.IRQ)
		tags[stageLabel] = "softIRQ"
		cc.mSystemCPUStat.Record(tags, c.SoftIRQ)
		tags[stageLabel] = "steal"
		cc.mSystemCPUStat.Record(tags, c.Steal)
		tags[stageLabel] = "guest"
		cc.mSystemCPUStat.Record(tags, c.Guest)
		tags[stageLabel] = "guestNice"
		cc.mSystemCPUStat.Record(tags, c.GuestNice)
	}
}

func (cc *cpuCollector) collect() {
	if cc == nil {
		return
	}

	cc.recordLoad()
	cc.recordUsage()
	cc.recordSystemStats()
}
