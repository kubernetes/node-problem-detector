/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package apiservermonitor

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/node-problem-detector/pkg/types"
)

func TestApplyConfiguration(t *testing.T) {
	enableMetrics := true
	disableMetrics := false
	skipInitial := false
	enableSkipInitial := true

	testCases := []struct {
		name           string
		originalConfig MonitorConfig
		wantedConfig   MonitorConfig
		isError        bool
	}{
		{
			name:           "defaults",
			originalConfig: MonitorConfig{},
			wantedConfig: MonitorConfig{
				InvokeIntervalString: defaultInvokeIntervalString,
				InvokeInterval:       defaultInvokeInterval,
				TimeoutString:        defaultTimeoutString,
				Timeout:              defaultTimeout,
				Source:               defaultSource,
				DefaultConditions: []types.Condition{
					{
						Type:    defaultConditionType,
						Reason:  defaultConditionReason,
						Message: defaultConditionMessage,
					},
				},
				EnableMetricsReporting: &enableMetrics,
				SkipInitialStatus:      &skipInitial,
			},
			isError: false,
		},
		{
			name: "custom settings",
			originalConfig: MonitorConfig{
				InvokeIntervalString: "30s",
				TimeoutString:        "10s",
				Source:               "custom-apiserver-monitor",
				DefaultConditions: []types.Condition{
					{
						Type:    "CustomUnreachable",
						Reason:  "CustomReachable",
						Message: "custom apiserver reachable",
					},
				},
				EnableMetricsReporting: &disableMetrics,
				SkipInitialStatus:      &enableSkipInitial,
			},
			wantedConfig: MonitorConfig{
				InvokeIntervalString: "30s",
				InvokeInterval:       30 * time.Second,
				TimeoutString:        "10s",
				Timeout:              10 * time.Second,
				Source:               "custom-apiserver-monitor",
				DefaultConditions: []types.Condition{
					{
						Type:    "CustomUnreachable",
						Reason:  "CustomReachable",
						Message: "custom apiserver reachable",
					},
				},
				EnableMetricsReporting: &disableMetrics,
				SkipInitialStatus:      &enableSkipInitial,
			},
			isError: false,
		},
		{
			name: "invalid invokeInterval",
			originalConfig: MonitorConfig{
				InvokeIntervalString: "invalid",
			},
			isError: true,
		},
		{
			name: "invalid timeout",
			originalConfig: MonitorConfig{
				TimeoutString: "invalid",
			},
			isError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := test.originalConfig.ApplyConfiguration()
			if err == nil && test.isError {
				t.Errorf("Wanted an error, got nil")
			}
			if err != nil && !test.isError {
				t.Errorf("Wanted nil, got error: %v", err)
			}
			if !test.isError && !reflect.DeepEqual(test.originalConfig, test.wantedConfig) {
				t.Errorf("Wanted: %+v, got: %+v", test.wantedConfig, test.originalConfig)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name    string
		config  MonitorConfig
		isError bool
	}{
		{
			name: "valid default config",
			config: MonitorConfig{
				InvokeIntervalString: "10s",
				TimeoutString:        "5s",
			},
			isError: false,
		},
		{
			name: "non-positive invokeInterval",
			config: MonitorConfig{
				InvokeIntervalString: "-1s",
				TimeoutString:        "5s",
			},
			isError: true,
		},
		{
			name: "non-positive timeout",
			config: MonitorConfig{
				InvokeIntervalString: "10s",
				TimeoutString:        "-1s",
			},
			isError: true,
		},
		{
			name: "timeout greater than invokeInterval",
			config: MonitorConfig{
				InvokeIntervalString: "5s",
				TimeoutString:        "10s",
			},
			isError: true,
		},
		{
			name: "empty condition type",
			config: MonitorConfig{
				InvokeIntervalString: "10s",
				TimeoutString:        "5s",
				DefaultConditions: []types.Condition{
					{
						Type:    "",
						Reason:  "SomeReason",
						Message: "some message",
					},
				},
			},
			isError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if err := test.config.ApplyConfiguration(); err != nil {
				t.Fatalf("ApplyConfiguration() unexpected error: %v", err)
			}
			err := test.config.Validate()
			if test.isError && err == nil {
				t.Errorf("Validate() = nil, want error for config %+v", test.config)
			}
			if !test.isError && err != nil {
				t.Errorf("Validate() = %v, want nil for config %+v", err, test.config)
			}
		})
	}
}
