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
	"io/ioutil"
	"time"

	"github.com/golang/glog"

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
	config        ssmtypes.SystemStatsConfig
	diskCollector *diskCollector
	tomb          *tomb.Tomb
}

// NewSystemStatsMonitorOrDie creates a system stats monitor.
func NewSystemStatsMonitorOrDie(configPath string) types.Monitor {
	ssm := systemStatsMonitor{
		tomb: tomb.NewTomb(),
	}

	// Apply configurations.
	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		glog.Fatalf("Failed to read configuration file %q: %v", configPath, err)
	}
	err = json.Unmarshal(f, &ssm.config)
	if err != nil {
		glog.Fatalf("Failed to unmarshal configuration file %q: %v", configPath, err)
	}

	err = ssm.config.ApplyConfiguration()
	if err != nil {
		glog.Fatalf("Failed to apply configuration for %q: %v", configPath, err)
	}

	err = ssm.config.Validate()
	if err != nil {
		glog.Fatalf("Failed to validate configuration %+v: %v", ssm.config, err)
	}

	// Initialize diskCollector if needed.
	if len(ssm.config.DiskConfig.MetricsConfigs) > 0 {
		ssm.diskCollector = NewDiskCollectorOrDie(&ssm.config.DiskConfig)
	}
	return &ssm
}

func (ssm *systemStatsMonitor) Start() (<-chan *types.Status, error) {
	glog.Info("Start system stats monitor")
	go ssm.monitorLoop()
	return nil, nil
}

func (ssm *systemStatsMonitor) monitorLoop() {
	defer ssm.tomb.Done()

	runTicker := time.NewTicker(ssm.config.InvokeInterval)
	defer runTicker.Stop()

	select {
	case <-ssm.tomb.Stopping():
		glog.Infof("System stats monitor stopped")
		return
	default:
		ssm.diskCollector.collect()
	}

	for {
		select {
		case <-runTicker.C:
			ssm.diskCollector.collect()
		case <-ssm.tomb.Stopping():
			glog.Infof("System stats monitor stopped")
			return
		}
	}
}

func (ssm *systemStatsMonitor) Stop() {
	glog.Info("Stop system stats monitor")
	ssm.tomb.Stop()
}
