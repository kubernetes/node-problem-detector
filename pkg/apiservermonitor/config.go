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
	"fmt"
	"time"

	"k8s.io/node-problem-detector/pkg/types"
)

const (
	defaultInvokeInterval = 10 * time.Second
	defaultTimeout        = 5 * time.Second
	defaultSource         = "apiserver-monitor"

	defaultConditionType    = "APIServerUnreachable"
	defaultConditionReason  = "APIServerIsReachable"
	defaultConditionMessage = "apiserver is reachable"
)

var (
	defaultInvokeIntervalString   = defaultInvokeInterval.String()
	defaultTimeoutString          = defaultTimeout.String()
	defaultEnableMetricsReporting = true
	defaultSkipInitialStatus      = false
)

// MonitorConfig is the configuration of the apiserver monitor.
type MonitorConfig struct {
	// InvokeIntervalString is the interval string at which the apiserver check will be invoked.
	InvokeIntervalString string `json:"invokeInterval,omitempty"`
	// InvokeInterval is the interval at which the apiserver check will be invoked.
	InvokeInterval time.Duration `json:"-"`
	// TimeoutString is the timeout string for each apiserver check.
	TimeoutString string `json:"timeout,omitempty"`
	// Timeout is the timeout for each apiserver check.
	Timeout time.Duration `json:"-"`
	// Source is the source name of the apiserver monitor.
	Source string `json:"source,omitempty"`
	// DefaultConditions are the default states of all the conditions the apiserver monitor handles.
	DefaultConditions []types.Condition `json:"conditions,omitempty"`
	// EnableMetricsReporting describes whether to report problems as metrics or not.
	EnableMetricsReporting *bool `json:"metricsReporting,omitempty"`
	// SkipInitialStatus prevents the first status update with default conditions.
	SkipInitialStatus *bool `json:"skipInitialStatus,omitempty"`
}

// ApplyConfiguration applies default configurations.
func (mc *MonitorConfig) ApplyConfiguration() error {
	if mc.InvokeIntervalString == "" {
		mc.InvokeIntervalString = defaultInvokeIntervalString
	}
	if mc.TimeoutString == "" {
		mc.TimeoutString = defaultTimeoutString
	}
	if mc.Source == "" {
		mc.Source = defaultSource
	}
	if len(mc.DefaultConditions) == 0 {
		mc.DefaultConditions = []types.Condition{
			{
				Type:    defaultConditionType,
				Reason:  defaultConditionReason,
				Message: defaultConditionMessage,
			},
		}
	}
	if mc.EnableMetricsReporting == nil {
		enableMetricsReporting := defaultEnableMetricsReporting
		mc.EnableMetricsReporting = &enableMetricsReporting
	}
	if mc.SkipInitialStatus == nil {
		skipInitialStatus := defaultSkipInitialStatus
		mc.SkipInitialStatus = &skipInitialStatus
	}

	var err error
	mc.InvokeInterval, err = time.ParseDuration(mc.InvokeIntervalString)
	if err != nil {
		return fmt.Errorf("error in parsing invokeInterval %q: %w", mc.InvokeIntervalString, err)
	}
	mc.Timeout, err = time.ParseDuration(mc.TimeoutString)
	if err != nil {
		return fmt.Errorf("error in parsing timeout %q: %w", mc.TimeoutString, err)
	}

	return nil
}

// Validate verifies whether the settings in MonitorConfig are valid.
func (mc *MonitorConfig) Validate() error {
	if mc.InvokeInterval <= 0 {
		return fmt.Errorf("invokeInterval %v must be above 0s", mc.InvokeInterval)
	}
	if mc.Timeout <= 0 {
		return fmt.Errorf("timeout %v must be above 0s", mc.Timeout)
	}
	if mc.Timeout > mc.InvokeInterval {
		return fmt.Errorf("timeout %v must not be greater than invokeInterval %v", mc.Timeout, mc.InvokeInterval)
	}
	for _, cond := range mc.DefaultConditions {
		if cond.Type == "" {
			return fmt.Errorf("condition type must not be empty: %+v", cond)
		}
	}
	return nil
}
