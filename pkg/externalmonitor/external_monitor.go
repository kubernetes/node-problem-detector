/*
Copyright The Kubernetes Authors All rights reserved.

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

package externalmonitor

import (
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/pkg/externalmonitor/types"
	"k8s.io/node-problem-detector/pkg/problemdaemon"
	npdt "k8s.io/node-problem-detector/pkg/types"
)

const (
	// MonitorName is the name used for registering the external monitor.
	MonitorName = "external-monitor"
)

func init() {
	problemdaemon.Register(
		MonitorName,
		npdt.ProblemDaemonHandler{
			CreateProblemDaemonOrDie: NewExternalMonitorOrDie,
			CmdOptionDescription:     "Set to external monitor config file paths.",
		})
}

// NewExternalMonitorOrDie creates a new external monitor from the config file path.
// This function follows the same pattern as other monitors in NPD.
func NewExternalMonitorOrDie(configPath string) npdt.Monitor {
	klog.Infof("Creating external monitor from config: %s", configPath)

	config, err := LoadConfiguration(configPath)
	if err != nil {
		klog.Fatalf("Failed to load external monitor configuration from %s: %v", configPath, err)
	}

	if err := config.ApplyConfiguration(); err != nil {
		klog.Fatalf("Failed to apply external monitor configuration: %v", err)
	}

	if err := config.Validate(); err != nil {
		klog.Fatalf("Invalid external monitor configuration: %v", err)
	}

	monitor, err := NewExternalMonitorProxy(config)
	if err != nil {
		klog.Fatalf("Failed to create external monitor proxy: %v", err)
	}

	klog.Infof("Created external monitor: %s (socket: %s)",
		config.Source, config.PluginConfig.SocketAddress)

	return monitor
}

// LoadConfiguration loads and parses the external monitor configuration from a file.
func LoadConfiguration(configPath string) (*types.ExternalMonitorConfig, error) {
	// Read configuration file (reusing pattern from custompluginmonitor)
	configBytes, err := readFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", configPath, err)
	}

	// Parse JSON configuration
	var config types.ExternalMonitorConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %v", err)
	}

	return &config, nil
}

// readFile reads the content of a file - abstracted for testing.
var readFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}
