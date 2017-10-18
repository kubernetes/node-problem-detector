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

package systemlogmonitor

import (
	"regexp"

	watchertypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	systemlogtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/types"
)

// MonitorConfig is the configuration of log monitor.
type MonitorConfig struct {
	// WatcherConfig is the configuration of log watcher.
	watchertypes.WatcherConfig
	// BufferSize is the size (in lines) of the log buffer.
	BufferSize int `json:"bufferSize"`
	// Source is the source name of the log monitor
	Source string `json:"source"`
	// DefaultConditions are the default states of all the conditions log monitor should handle.
	DefaultConditions []types.Condition `json:"conditions"`
	// Rules are the rules log monitor will follow to parse the log file.
	Rules []systemlogtypes.Rule `json:"rules"`
}

// ApplyConfiguration applies default configurations.
func (mc *MonitorConfig) ApplyDefaultConfiguration() {
	if mc.BufferSize == 0 {
		mc.BufferSize = 10
	}
	if mc.WatcherConfig.Lookback == "" {
		mc.WatcherConfig.Lookback = "0"
	}
}

// ValidateRules verifies whether the regular expressions in the rules are valid.
func (mc MonitorConfig) ValidateRules() error {
	for _, rule := range mc.Rules {
		_, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return err
		}
	}
	return nil
}
