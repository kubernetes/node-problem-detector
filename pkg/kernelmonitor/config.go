/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package kernelmonitor

import (
	watchertypes "k8s.io/node-problem-detector/pkg/kernelmonitor/logwatchers/types"
	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/types"
)

// MonitorConfig is the configuration of kernel monitor.
type MonitorConfig struct {
	// WatcherConfig is the configuration of kernel log watcher.
	watchertypes.WatcherConfig
	// BufferSize is the size (in lines) of the log buffer.
	BufferSize int `json:"bufferSize"`
	// Source is the source name of the kernel monitor
	Source string `json:"source"`
	// DefaultConditions are the default states of all the conditions kernel monitor should handle.
	DefaultConditions []types.Condition `json:"conditions"`
	// Rules are the rules kernel monitor will follow to parse the log file.
	Rules []kerntypes.Rule `json:"rules"`
}

// applyDefaultConfiguration applies default configurations.
func applyDefaultConfiguration(cfg *MonitorConfig) {
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 10
	}
	if cfg.WatcherConfig.Lookback == "" {
		cfg.WatcherConfig.Lookback = "0"
	}
}
