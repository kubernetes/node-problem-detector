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
	"reflect"
	"testing"
	"time"
)

func TestApplyConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		orignalConfig SystemStatsConfig
		wantedConfig  SystemStatsConfig
		isError       bool
	}{
		{
			name: "normal",
			orignalConfig: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{
					LsblkTimeoutString: "5s",
				},
				InvokeIntervalString: "60s",
			},
			isError: false,
			wantedConfig: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{
					LsblkTimeout:       5 * time.Second,
					LsblkTimeoutString: "5s",
				},
				OsFeatureConfig: OSFeatureStatsConfig{
					KnownModulesConfigPath: "config/guestosconfig/known-modules.json",
				},
				InvokeIntervalString: "60s",
				InvokeInterval:       60 * time.Second,
			},
		},
		{
			name: "empty",
			orignalConfig: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{},
			},
			isError: false,
			wantedConfig: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{
					LsblkTimeout:       5 * time.Second,
					LsblkTimeoutString: "5s",
				},
				OsFeatureConfig: OSFeatureStatsConfig{
					KnownModulesConfigPath: "config/guestosconfig/known-modules.json",
				},
				InvokeIntervalString: "1m0s",
				InvokeInterval:       60 * time.Second,
			},
		},
		{
			name: "error",
			orignalConfig: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{
					LsblkTimeoutString: "foo",
				},
			},
			isError: true,
			wantedConfig: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{},
				OsFeatureConfig: OSFeatureStatsConfig{
					KnownModulesConfigPath: "config/guestosconfig/known-modules.json",
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := test.orignalConfig.ApplyConfiguration()
			if err == nil && test.isError {
				t.Errorf("Wanted an error got nil")
			}
			if err != nil && !test.isError {
				t.Errorf("Wanted nil got an error")
			}
			if !test.isError && !reflect.DeepEqual(test.orignalConfig, test.wantedConfig) {
				t.Errorf("Wanted: %+v. \nGot: %+v", test.wantedConfig, test.orignalConfig)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		config  SystemStatsConfig
		isError bool
	}{
		{
			name: "normal",
			config: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{
					LsblkTimeoutString: "5s",
				},
				InvokeIntervalString: "60s",
			},
			isError: false,
		},
		{
			name: "negative-invoke-interval",
			config: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{
					LsblkTimeoutString: "5s",
				},
				InvokeIntervalString: "-1s",
			},
			isError: true,
		},
		{
			name: "negative-lsblk-timeout",
			config: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{
					LsblkTimeoutString: "-1s",
				},
				InvokeIntervalString: "60s",
			},
			isError: true,
		},
		{
			name: "lsblk-timeout-bigger-than-invoke-interval",
			config: SystemStatsConfig{
				DiskConfig: DiskStatsConfig{
					LsblkTimeoutString: "90s",
				},
				InvokeIntervalString: "60s",
			},
			isError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.config.ApplyConfiguration(); err != nil {
				t.Errorf("Wanted no error with config %+v, got %v", test.config, err)
			}

			err := test.config.Validate()
			if test.isError && err == nil {
				t.Errorf("Wanted an error with config %+v, got nil", test.config)
			}
			if !test.isError && err != nil {
				t.Errorf("Wanted nil with config %+v got an error", test.config)
			}
		})
	}
}
