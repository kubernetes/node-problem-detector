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
	"github.com/shirou/gopsutil/v4/load"
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
		if err := cc.mRunnableTaskCount.Record(map[string]string{}, loadAvg.Load1); err != nil {
			klog.Errorf("Failed to record runnable task count: %v", err)
		}
	}
	if cc.mCpuLoad1m != nil {
		if err := cc.mCpuLoad1m.Record(map[string]string{}, loadAvg.Load1); err != nil {
			klog.Errorf("Failed to record cpu load 1m: %v", err)
		}
	}
	if cc.mCpuLoad5m != nil {
		if err := cc.mCpuLoad5m.Record(map[string]string{}, loadAvg.Load5); err != nil {
			klog.Errorf("Failed to record cpu load 5m: %v", err)
		}
	}
	if cc.mCpuLoad15m != nil {
		if err := cc.mCpuLoad15m.Record(map[string]string{}, loadAvg.Load15); err != nil {
			klog.Errorf("Failed to record cpu load 15m: %v", err)
		}
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
	if err != nil {
		klog.Errorf("Failed to open procfs: %v", err)
		return
	}
	stats, err := fs.Stat()
	if err != nil {
		klog.Errorf("Failed to retrieve cpu/process stats: %v", err)
		return
	}

	if cc.mSystemProcessesTotal != nil {
		if err := cc.mSystemProcessesTotal.Record(map[string]string{}, int64(stats.ProcessCreated)); err != nil {
			klog.Errorf("Failed to record system processes total: %v", err)
		}
	}

	if cc.mSystemProcsRunning != nil {
		if err := cc.mSystemProcsRunning.Record(map[string]string{}, int64(stats.ProcessesRunning)); err != nil {
			klog.Errorf("Failed to record system procs running: %v", err)
		}
	}

	if cc.mSystemProcsBlocked != nil {
		if err := cc.mSystemProcsBlocked.Record(map[string]string{}, int64(stats.ProcessesBlocked)); err != nil {
			klog.Errorf("Failed to record system procs blocked: %v", err)
		}
	}

	if cc.mSystemInterruptsTotal != nil {
		if err := cc.mSystemInterruptsTotal.Record(map[string]string{}, int64(stats.IRQTotal)); err != nil {
			klog.Errorf("Failed to record system interrupts total: %v", err)
		}
	}

	if cc.mSystemCPUStat != nil {
		for i, c := range stats.CPU {
			tags := map[string]string{}
			tags[cpuLabel] = fmt.Sprintf("cpu%d", i)

			tags[stageLabel] = "user"
			if err := cc.mSystemCPUStat.Record(tags, c.User); err != nil {
				klog.Errorf("Failed to record system cpu stat for user: %v", err)
			}
			tags[stageLabel] = "nice"
			if err := cc.mSystemCPUStat.Record(tags, c.Nice); err != nil {
				klog.Errorf("Failed to record system cpu stat for nice: %v", err)
			}
			tags[stageLabel] = "system"
			if err := cc.mSystemCPUStat.Record(tags, c.System); err != nil {
				klog.Errorf("Failed to record system cpu stat for system: %v", err)
			}
			tags[stageLabel] = "idle"
			if err := cc.mSystemCPUStat.Record(tags, c.Idle); err != nil {
				klog.Errorf("Failed to record system cpu stat for idle: %v", err)
			}
			tags[stageLabel] = "iowait"
			if err := cc.mSystemCPUStat.Record(tags, c.Iowait); err != nil {
				klog.Errorf("Failed to record system cpu stat for iowait: %v", err)
			}
			tags[stageLabel] = "iRQ"
			if err := cc.mSystemCPUStat.Record(tags, c.IRQ); err != nil {
				klog.Errorf("Failed to record system cpu stat for iRQ: %v", err)
			}
			tags[stageLabel] = "softIRQ"
			if err := cc.mSystemCPUStat.Record(tags, c.SoftIRQ); err != nil {
				klog.Errorf("Failed to record system cpu stat for softIRQ: %v", err)
			}
			tags[stageLabel] = "steal"
			if err := cc.mSystemCPUStat.Record(tags, c.Steal); err != nil {
				klog.Errorf("Failed to record system cpu stat for steal: %v", err)
			}
			tags[stageLabel] = "guest"
			if err := cc.mSystemCPUStat.Record(tags, c.Guest); err != nil {
				klog.Errorf("Failed to record system cpu stat for guest: %v", err)
			}
			tags[stageLabel] = "guestNice"
			if err := cc.mSystemCPUStat.Record(tags, c.GuestNice); err != nil {
				klog.Errorf("Failed to record system cpu stat for guestNice: %v", err)
			}
		}
	}
}
