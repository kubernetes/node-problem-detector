//go:build unix

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

	"github.com/prometheus/procfs"
	"github.com/shirou/gopsutil/v3/load"
	"k8s.io/klog/v2"
)

func (cc *cpuCollector) recordLoad() {
	// don't collect the load metrics if the configs are not present.
	if cc.mRunnableTaskCount == nil &&
		cc.mCpuLoad15m == nil && cc.mCpuLoad1m == nil && cc.mCpuLoad5m == nil {
		return
	}

	loadAvg, err := load.Avg()
	if err != nil {
		klog.Errorf("Failed to retrieve average CPU load: %v", err)
		return
	}

	if cc.mRunnableTaskCount != nil {
		cc.mRunnableTaskCount.Record(map[string]string{}, loadAvg.Load1)
	}
	if cc.mCpuLoad1m != nil {
		cc.mCpuLoad1m.Record(map[string]string{}, loadAvg.Load1)
	}
	if cc.mCpuLoad5m != nil {
		cc.mCpuLoad5m.Record(map[string]string{}, loadAvg.Load5)
	}
	if cc.mCpuLoad15m != nil {
		cc.mCpuLoad15m.Record(map[string]string{}, loadAvg.Load15)
	}
}

func (cc *cpuCollector) recordSystemStats() {
	// don't collect the load metrics if the configs are not present.
	if cc.mSystemCPUStat == nil && cc.mSystemInterruptsTotal == nil &&
		cc.mSystemProcessesTotal == nil && cc.mSystemProcsBlocked == nil &&
		cc.mSystemProcsRunning == nil {
		return
	}

	fs, err := procfs.NewFS(cc.procPath)
	stats, err := fs.Stat()
	if err != nil {
		klog.Errorf("Failed to retrieve cpu/process stats: %v", err)
		return
	}

	if cc.mSystemProcessesTotal != nil {
		cc.mSystemProcessesTotal.Record(map[string]string{}, int64(stats.ProcessCreated))
	}

	if cc.mSystemProcsRunning != nil {
		cc.mSystemProcsRunning.Record(map[string]string{}, int64(stats.ProcessesRunning))
	}

	if cc.mSystemProcsBlocked != nil {
		cc.mSystemProcsBlocked.Record(map[string]string{}, int64(stats.ProcessesBlocked))
	}

	if cc.mSystemInterruptsTotal != nil {
		cc.mSystemInterruptsTotal.Record(map[string]string{}, int64(stats.IRQTotal))
	}

	if cc.mSystemCPUStat != nil {
		for i, c := range stats.CPU {
			tags := map[string]string{}
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
}
