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
	"github.com/shirou/gopsutil/v4/cpu"
	"k8s.io/klog/v2"

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

	config   *ssmtypes.CPUStatsConfig
	procPath string

	lastUsageTime map[string]float64
}

func NewCPUCollectorOrDie(cpuConfig *ssmtypes.CPUStatsConfig, procPath string) *cpuCollector {
	cc := cpuCollector{
		config:   cpuConfig,
		procPath: procPath,
	}

	var err error
	cc.mRunnableTaskCount, err = metrics.NewFloat64Metric(
		metrics.CPURunnableTaskCountID,
		cpuConfig.MetricsConfigs[string(metrics.CPURunnableTaskCountID)].DisplayName,
		"The average number of runnable tasks in the run-queue during the last minute",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.CPURunnableTaskCountID, err)
	}

	cc.mUsageTime, err = metrics.NewFloat64Metric(
		metrics.CPUUsageTimeID,
		cpuConfig.MetricsConfigs[string(metrics.CPUUsageTimeID)].DisplayName,
		"CPU usage, in seconds",
		"s",
		metrics.Sum,
		[]string{stateLabel})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.CPUUsageTimeID, err)
	}

	cc.mCpuLoad1m, err = metrics.NewFloat64Metric(
		metrics.CPULoad1m,
		cpuConfig.MetricsConfigs[string(metrics.CPULoad1m)].DisplayName,
		"CPU average load (1m)",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.CPULoad1m, err)
	}

	cc.mCpuLoad5m, err = metrics.NewFloat64Metric(
		metrics.CPULoad5m,
		cpuConfig.MetricsConfigs[string(metrics.CPULoad5m)].DisplayName,
		"CPU average load (5m)",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.CPULoad5m, err)
	}

	cc.mCpuLoad15m, err = metrics.NewFloat64Metric(
		metrics.CPULoad15m,
		cpuConfig.MetricsConfigs[string(metrics.CPULoad15m)].DisplayName,
		"CPU average load (15m)",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.CPULoad15m, err)
	}

	cc.mSystemProcessesTotal, err = metrics.NewInt64Metric(
		metrics.SystemProcessesTotal,
		cpuConfig.MetricsConfigs[string(metrics.SystemProcessesTotal)].DisplayName,
		"Number of forks since boot.",
		"1",
		metrics.Sum,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.SystemProcessesTotal, err)
	}

	cc.mSystemProcsRunning, err = metrics.NewInt64Metric(
		metrics.SystemProcsRunning,
		cpuConfig.MetricsConfigs[string(metrics.SystemProcsRunning)].DisplayName,
		"Number of processes currently running.",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.SystemProcsRunning, err)
	}

	cc.mSystemProcsBlocked, err = metrics.NewInt64Metric(
		metrics.SystemProcsBlocked,
		cpuConfig.MetricsConfigs[string(metrics.SystemProcsBlocked)].DisplayName,
		"Number of processes currently blocked.",
		"1",
		metrics.LastValue,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.SystemProcsBlocked, err)
	}

	cc.mSystemInterruptsTotal, err = metrics.NewInt64Metric(
		metrics.SystemInterruptsTotal,
		cpuConfig.MetricsConfigs[string(metrics.SystemInterruptsTotal)].DisplayName,
		"Total number of interrupts serviced (cumulative).",
		"1",
		metrics.Sum,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.SystemInterruptsTotal, err)
	}

	cc.mSystemCPUStat, err = metrics.NewFloat64Metric(
		metrics.SystemCPUStat,
		cpuConfig.MetricsConfigs[string(metrics.SystemCPUStat)].DisplayName,
		"Cumulative time each cpu spent in various stages.",
		"ns",
		metrics.Sum,
		[]string{cpuLabel, stageLabel})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.SystemCPUStat, err)
	}

	cc.lastUsageTime = make(map[string]float64)

	return &cc
}

func (cc *cpuCollector) recordUsage() {
	if cc.mUsageTime == nil {
		return
	}

	// Set percpu=false to get aggregated usage from all CPUs.
	timersStats, err := cpu.Times(false)
	if err != nil {
		klog.Errorf("Failed to retrieve CPU timers stat: %v", err)
		return
	}
	timersStat := timersStats[0]

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "user"}, clockTick*timersStat.User-cc.lastUsageTime["user"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for user: %v", err)
	}
	cc.lastUsageTime["user"] = clockTick * timersStat.User

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "system"}, clockTick*timersStat.System-cc.lastUsageTime["system"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for system: %v", err)
	}
	cc.lastUsageTime["system"] = clockTick * timersStat.System

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "idle"}, clockTick*timersStat.Idle-cc.lastUsageTime["idle"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for idle: %v", err)
	}
	cc.lastUsageTime["idle"] = clockTick * timersStat.Idle

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "nice"}, clockTick*timersStat.Nice-cc.lastUsageTime["nice"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for nice: %v", err)
	}
	cc.lastUsageTime["nice"] = clockTick * timersStat.Nice

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "iowait"}, clockTick*timersStat.Iowait-cc.lastUsageTime["iowait"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for iowait: %v", err)
	}
	cc.lastUsageTime["iowait"] = clockTick * timersStat.Iowait

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "irq"}, clockTick*timersStat.Irq-cc.lastUsageTime["irq"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for irq: %v", err)
	}
	cc.lastUsageTime["irq"] = clockTick * timersStat.Irq

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "softirq"}, clockTick*timersStat.Softirq-cc.lastUsageTime["softirq"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for softirq: %v", err)
	}
	cc.lastUsageTime["softirq"] = clockTick * timersStat.Softirq

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "steal"}, clockTick*timersStat.Steal-cc.lastUsageTime["steal"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for steal: %v", err)
	}
	cc.lastUsageTime["steal"] = clockTick * timersStat.Steal

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "guest"}, clockTick*timersStat.Guest-cc.lastUsageTime["guest"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for guest: %v", err)
	}
	cc.lastUsageTime["guest"] = clockTick * timersStat.Guest

	if err := cc.mUsageTime.Record(map[string]string{stateLabel: "guest_nice"}, clockTick*timersStat.GuestNice-cc.lastUsageTime["guest_nice"]); err != nil {
		klog.Errorf("Failed to record cpu usage time for guest_nice: %v", err)
	}
	cc.lastUsageTime["guest_nice"] = clockTick * timersStat.GuestNice
}

func (cc *cpuCollector) collect() {
	if cc == nil {
		return
	}

	cc.recordLoad()
	cc.recordUsage()
	cc.recordSystemStats()
}
