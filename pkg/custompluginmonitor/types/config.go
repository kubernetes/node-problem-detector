/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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
	"os"
	"time"

	"k8s.io/node-problem-detector/pkg/types"
)

var (
	defaultGlobalTimeout                     = 5 * time.Second
	defaultGlobalTimeoutString               = defaultGlobalTimeout.String()
	defaultInvokeInterval                    = 30 * time.Second
	defaultInvokeIntervalString              = defaultInvokeInterval.String()
	defaultMaxOutputLength                   = 80
	defaultConcurrency                       = 3
	defaultMessageChangeBasedConditionUpdate = false

	customPluginName = "custom"
)

type pluginGlobalConfig struct {
	// InvokeIntervalString is the interval string at which plugins will be invoked.
	InvokeIntervalString *string `json:"invoke_interval,omitempty"`
	// TimeoutString is the global plugin execution timeout string.
	TimeoutString *string `json:"timeout,omitempty"`
	// InvokeInterval is the interval at which plugins will be invoked.
	InvokeInterval *time.Duration `json:"-"`
	// Timeout is the global plugin execution timeout.
	Timeout *time.Duration `json:"-"`
	// MaxOutputLength is the maximum plugin output message length.
	MaxOutputLength *int `json:"max_output_length,omitempty"`
	// Concurrency is the number of concurrent running plugins.
	Concurrency *int `json:"concurrency,omitempty"`
	// EnableMessageChangeBasedConditionUpdate indicates whether NPD should enable message change based condition update.
	EnableMessageChangeBasedConditionUpdate *bool `json:"enable_message_change_based_condition_update,omitempty"`
}

// Custom plugin config is the configuration of custom plugin monitor.
type CustomPluginConfig struct {
	// Plugin is the name of plugin which is currently used.
	// Currently supported: custom.
	Plugin string `json:"plugin,omitempty"`
	// PluginConfig is global plugin configuration.
	PluginGlobalConfig pluginGlobalConfig `json:"pluginConfig,omitempty"`
	// Source is the source name of the custom plugin monitor
	Source string `json:"source"`
	// DefaultConditions are the default states of all the conditions custom plugin monitor should handle.
	DefaultConditions []types.Condition `json:"conditions"`
	// Rules are the rules custom plugin monitor will follow to parse and invoke plugins.
	Rules []*CustomRule `json:"rules"`
}

// ApplyConfiguration applies default configurations.
func (cpc *CustomPluginConfig) ApplyConfiguration() error {
	if cpc.PluginGlobalConfig.TimeoutString == nil {
		cpc.PluginGlobalConfig.TimeoutString = &defaultGlobalTimeoutString
	}

	timeout, err := time.ParseDuration(*cpc.PluginGlobalConfig.TimeoutString)
	if err != nil {
		return fmt.Errorf("error in parsing global timeout %q: %v", *cpc.PluginGlobalConfig.TimeoutString, err)
	}

	cpc.PluginGlobalConfig.Timeout = &timeout

	if cpc.PluginGlobalConfig.InvokeIntervalString == nil {
		cpc.PluginGlobalConfig.InvokeIntervalString = &defaultInvokeIntervalString
	}

	invokeInterval, err := time.ParseDuration(*cpc.PluginGlobalConfig.InvokeIntervalString)
	if err != nil {
		return fmt.Errorf("error in parsing invoke interval %q: %v", *cpc.PluginGlobalConfig.InvokeIntervalString, err)
	}

	cpc.PluginGlobalConfig.InvokeInterval = &invokeInterval

	if cpc.PluginGlobalConfig.MaxOutputLength == nil {
		cpc.PluginGlobalConfig.MaxOutputLength = &defaultMaxOutputLength
	}
	if cpc.PluginGlobalConfig.Concurrency == nil {
		cpc.PluginGlobalConfig.Concurrency = &defaultConcurrency
	}
	if cpc.PluginGlobalConfig.EnableMessageChangeBasedConditionUpdate == nil {
		cpc.PluginGlobalConfig.EnableMessageChangeBasedConditionUpdate = &defaultMessageChangeBasedConditionUpdate
	}

	for _, rule := range cpc.Rules {
		if rule.TimeoutString != nil {
			timeout, err := time.ParseDuration(*rule.TimeoutString)
			if err != nil {
				return fmt.Errorf("error in parsing rule timeout %+v: %v", rule, err)
			}
			rule.Timeout = &timeout
		}
	}

	return nil
}

// Validate verifies whether the settings in CustomPluginConfig are valid.
func (cpc CustomPluginConfig) Validate() error {
	if cpc.Plugin != customPluginName {
		return fmt.Errorf("NPD does not support %q plugin for now. Only support \"custom\"", cpc.Plugin)
	}

	for _, rule := range cpc.Rules {
		if rule.Timeout != nil && *rule.Timeout > *cpc.PluginGlobalConfig.Timeout {
			return fmt.Errorf("plugin timeout is greater than global timeout. "+
				"Rule: %+v. Global timeout: %v", rule, cpc.PluginGlobalConfig.Timeout)
		}
	}

	for _, rule := range cpc.Rules {
		if _, err := os.Stat(rule.Path); os.IsNotExist(err) {
			return fmt.Errorf("rule path %q does not exist. Rule: %+v", rule.Path, rule)
		}
	}

	return nil
}
