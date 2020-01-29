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

package types

import (
	"fmt"
	"time"
)

var (
	defaultInvokeIntervalString = (60 * time.Second).String()
	defaultlsblkTimeoutString   = (5 * time.Second).String()
)

type MetricConfig struct {
	DisplayName string `json:"displayName"`
}

type CPUStatsConfig struct {
	MetricsConfigs map[string]MetricConfig `json:"metricsConfigs"`
}

type DiskStatsConfig struct {
	MetricsConfigs        map[string]MetricConfig `json:"metricsConfigs"`
	IncludeRootBlk        bool                    `json:"includeRootBlk"`
	IncludeAllAttachedBlk bool                    `json:"includeAllAttachedBlk"`
	LsblkTimeoutString    string                  `json:"lsblkTimeout"`
	LsblkTimeout          time.Duration           `json:"-"`
}

type HostStatsConfig struct {
	MetricsConfigs map[string]MetricConfig `json:"metricsConfigs"`
}

type MemoryStatsConfig struct {
	MetricsConfigs map[string]MetricConfig `json:"metricsConfigs"`
}

type SystemStatsConfig struct {
	CPUConfig            CPUStatsConfig    `json:"cpu"`
	DiskConfig           DiskStatsConfig   `json:"disk"`
	HostConfig           HostStatsConfig   `json:"host"`
	MemoryConfig         MemoryStatsConfig `json:"memory"`
	InvokeIntervalString string            `json:"invokeInterval"`
	InvokeInterval       time.Duration     `json:"-"`
}

// ApplyConfiguration applies default configurations.
func (ssc *SystemStatsConfig) ApplyConfiguration() error {
	if ssc.InvokeIntervalString == "" {
		ssc.InvokeIntervalString = defaultInvokeIntervalString
	}
	if ssc.DiskConfig.LsblkTimeoutString == "" {
		ssc.DiskConfig.LsblkTimeoutString = defaultlsblkTimeoutString
	}

	var err error
	ssc.InvokeInterval, err = time.ParseDuration(ssc.InvokeIntervalString)
	if err != nil {
		return fmt.Errorf("error in parsing InvokeIntervalString %q: %v", ssc.InvokeIntervalString, err)
	}
	ssc.DiskConfig.LsblkTimeout, err = time.ParseDuration(ssc.DiskConfig.LsblkTimeoutString)
	if err != nil {
		return fmt.Errorf("error in parsing LsblkTimeoutString %q: %v", ssc.DiskConfig.LsblkTimeoutString, err)
	}

	return nil
}

// Validate verifies whether the settings are valid.
func (ssc *SystemStatsConfig) Validate() error {
	if ssc.InvokeInterval <= time.Duration(0) {
		return fmt.Errorf("InvokeInterval %v must be above 0s", ssc.InvokeInterval)
	}
	if ssc.DiskConfig.LsblkTimeout <= time.Duration(0) {
		return fmt.Errorf("LsblkTimeout %v must be above 0s", ssc.DiskConfig.LsblkTimeout)
	}
	if ssc.DiskConfig.LsblkTimeout > ssc.InvokeInterval {
		return fmt.Errorf("LsblkTimeout %v must be shorter than ssc.InvokeInterval %v", ssc.DiskConfig.LsblkTimeout, ssc.InvokeInterval)
	}

	return nil
}
