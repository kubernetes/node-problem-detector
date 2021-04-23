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
	"testing"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
)

const (
	fakeCPUConfig = `
{
	"metricsConfigs": {
		"cpu/load_15m": {
			"displayName": "cpu/load_15m"
		},
		"cpu/load_1m": {
			"displayName": "cpu/load_1m"
		},
		"cpu/load_5m": {
			"displayName": "cpu/load_5m"
		},
		"cpu/runnable_task_count": {
			"displayName": "cpu/runnable_task_count"
		},
		"cpu/usage_time": {
			"displayName": "cpu/usage_time"
		},
		"system/cpu_stat": {
			"displayName": "system/cpu_stat"
		},
		"system/interrupts_total": {
			"displayName": "system/interrupts_total"
		},
		"system/processes_total": {
			"displayName": "system/processes_total"
		},
		"system/procs_blocked": {
			"displayName": "system/procs_blocked"
		},
		"system/procs_running": {
			"displayName": "system/procs_running"
		}
	}
}
`
)

func TestCpuCollector(t *testing.T) {
	cfg := &ssmtypes.CPUStatsConfig{}
	if err := json.Unmarshal([]byte(fakeCPUConfig), cfg); err != nil {
		t.Fatalf("cannot load cpu config: %s", err)
	}
	mc := NewCPUCollectorOrDie(cfg)
	mc.collect()
}
