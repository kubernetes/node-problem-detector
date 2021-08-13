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
	process "github.com/shirou/gopsutil/process"
	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type processCollector struct {
	mRSSUsage *metrics.Int64Metric
	mVMUsage  *metrics.Int64Metric

	config *ssmtypes.ProcessStatsConfig
}

func NewProcessCollectorOrDie(processStatsConfig *ssmtypes.ProcessStatsConfig) *processCollector {
	pc := processCollector{config: processStatsConfig}
	var err error

	if processStatsConfig.MetricsConfigs[string(metrics.ProcessRSSUsage)].DisplayName != "" {
		pc.mRSSUsage, err = metrics.NewInt64Metric(
			metrics.ProcessRSSUsage,
			processStatsConfig.MetricsConfigs[string(metrics.ProcessRSSUsage)].DisplayName,
			"Resident memory usage of the given process.",
			"1",
			metrics.LastValue,
			[]string{pidLabel, processLabel, ownerLabel})
		if err != nil {
			glog.Fatalf("Error initializing metric for process/rss_usage: %v", err)
		}
	}

	if processStatsConfig.MetricsConfigs[string(metrics.ProcessVMSUsage)].DisplayName != "" {
		pc.mVMUsage, err = metrics.NewInt64Metric(
			metrics.ProcessVMSUsage,
			processStatsConfig.MetricsConfigs[string(metrics.ProcessVMSUsage)].DisplayName,
			"Resident memory usage of the given process.",
			"1",
			metrics.LastValue,
			[]string{pidLabel, processLabel, ownerLabel})
		if err != nil {
			glog.Fatalf("Error initializing metric for process/vm_usage: %v", err)
		}
	}

	return &pc
}

func (pc *processCollector) collect() {
	processesInfo, err := process.Processes()
	if err != nil {
		glog.Errorf("Failed to retrieve processes of the host: %v", err)
		return
	}

	for _, process := range processesInfo {
		pid := process.Pid
		processName, err := process.Name()
		memoryInfo, err := process.MemoryInfo()
		user, err := process.Username()

		if err != nil {
			glog.Infof("Failed to retrieve the process information: %v", err)
			return
		}

		pc.mRSSUsage.Record(map[string]string{pidLabel: fmt.Sprintf("%v", pid), processLabel: processName, ownerLabel: user}, int64(memoryInfo.RSS))
		pc.mVMUsage.Record(map[string]string{pidLabel: fmt.Sprintf("%v", pid), processLabel: processName, ownerLabel: user}, int64(memoryInfo.VMS))
	}

}
