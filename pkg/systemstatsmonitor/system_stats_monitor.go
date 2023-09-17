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
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/pkg/problemdaemon"
	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

const SystemStatsMonitorName = "system-stats-monitor"

func init() {
	problemdaemon.Register(SystemStatsMonitorName, types.ProblemDaemonHandler{
		CreateProblemDaemonOrDie: NewSystemStatsMonitorOrDie,
		CmdOptionDescription:     "Set to config file paths."})
}

type systemStatsMonitor struct {
	configPath         string
	config             ssmtypes.SystemStatsConfig
	cpuCollector       *cpuCollector
	diskCollector      *diskCollector
	hostCollector      *hostCollector
	memoryCollector    *memoryCollector
	netCollector       *netCollector
	osFeatureCollector *osFeatureCollector
	tomb               *tomb.Tomb
}

// NewSystemStatsMonitorOrDie creates a system stats monitor.
func NewSystemStatsMonitorOrDie(configPath string) types.Monitor {
	ssm := systemStatsMonitor{
		configPath: configPath,
		tomb:       tomb.NewTomb(),
	}

	// Apply configurations.
	f, err := os.ReadFile(configPath)
	if err != nil {
		klog.Fatalf("Failed to read configuration file %q: %v", configPath, err)
	}
	err = json.Unmarshal(f, &ssm.config)
	if err != nil {
		klog.Fatalf("Failed to unmarshal configuration file %q: %v", configPath, err)
	}

	err = ssm.config.ApplyConfiguration()
	if err != nil {
		klog.Fatalf("Failed to apply configuration for %q: %v", configPath, err)
	}

	err = ssm.config.Validate()
	if err != nil {
		klog.Fatalf("Failed to validate %s configuration %+v: %v", ssm.configPath, ssm.config, err)
	}

	if len(ssm.config.CPUConfig.MetricsConfigs) > 0 {
		ssm.cpuCollector = NewCPUCollectorOrDie(&ssm.config.CPUConfig, ssm.config.ProcPath)
	}
	if len(ssm.config.DiskConfig.MetricsConfigs) > 0 {
		ssm.diskCollector = NewDiskCollectorOrDie(&ssm.config.DiskConfig)
	}
	if len(ssm.config.HostConfig.MetricsConfigs) > 0 {
		ssm.hostCollector = NewHostCollectorOrDie(&ssm.config.HostConfig)
	}
	if len(ssm.config.MemoryConfig.MetricsConfigs) > 0 {
		ssm.memoryCollector = NewMemoryCollectorOrDie(&ssm.config.MemoryConfig)
	}
	if len(ssm.config.OsFeatureConfig.MetricsConfigs) > 0 {
		// update the KnownModulesConfigPath to relative the system-stats-monitors path
		// only when the KnownModulesConfigPath path is relative
		if !filepath.IsAbs(ssm.config.OsFeatureConfig.KnownModulesConfigPath) {
			ssm.config.OsFeatureConfig.KnownModulesConfigPath = filepath.Join(filepath.Dir(configPath),
				ssm.config.OsFeatureConfig.KnownModulesConfigPath)
		}
		ssm.osFeatureCollector = NewOsFeatureCollectorOrDie(&ssm.config.OsFeatureConfig, ssm.config.ProcPath)
	}
	if len(ssm.config.NetConfig.MetricsConfigs) > 0 {
		ssm.netCollector = NewNetCollectorOrDie(&ssm.config.NetConfig, ssm.config.ProcPath)
	}
	return &ssm
}

func (ssm *systemStatsMonitor) Start() (<-chan *types.Status, error) {
	klog.Infof("Start system stats monitor %s", ssm.configPath)
	go ssm.monitorLoop()
	return nil, nil
}

func (ssm *systemStatsMonitor) monitorLoop() {
	defer ssm.tomb.Done()

	runTicker := time.NewTicker(ssm.config.InvokeInterval)
	defer runTicker.Stop()

	select {
	case <-ssm.tomb.Stopping():
		klog.Infof("System stats monitor stopped: %s", ssm.configPath)
		return
	default:
		ssm.cpuCollector.collect()
		ssm.diskCollector.collect()
		ssm.hostCollector.collect()
		ssm.memoryCollector.collect()
		ssm.osFeatureCollector.collect()
		ssm.netCollector.collect()
	}

	for {
		select {
		case <-runTicker.C:
			ssm.cpuCollector.collect()
			ssm.diskCollector.collect()
			ssm.hostCollector.collect()
			ssm.memoryCollector.collect()
			ssm.osFeatureCollector.collect()
			ssm.netCollector.collect()
		case <-ssm.tomb.Stopping():
			klog.Infof("System stats monitor stopped: %s", ssm.configPath)
			return
		}
	}
}

func (ssm *systemStatsMonitor) Stop() {
	klog.Infof("Stop system stats monitor %s", ssm.configPath)
	ssm.tomb.Stop()
}
