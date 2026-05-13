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

package types

import (
	"fmt"
	"time"
)

// ExternalMonitorConfig contains configuration for external monitor plugins.
type ExternalMonitorConfig struct {
	// Plugin is the plugin type, must be "external".
	Plugin string `json:"plugin"`

	// PluginConfig contains external plugin specific configuration.
	PluginConfig ExternalPluginConfig `json:"pluginConfig"`

	// Source is the monitor source identifier.
	Source string `json:"source"`

	// MetricsReporting enables metrics reporting for this monitor.
	MetricsReporting bool `json:"metricsReporting,omitempty"`

	// Conditions define the possible conditions this monitor can report.
	Conditions []ConditionDefinition `json:"conditions,omitempty"`
}

// ExternalPluginConfig contains external plugin specific settings.
type ExternalPluginConfig struct {
	// SocketAddress is the Unix socket address for gRPC communication.
	SocketAddress string `json:"socketAddress"`

	// InvokeInterval is how often to call CheckHealth.
	InvokeInterval time.Duration `json:"invoke_interval"`

	// Timeout for each gRPC call.
	Timeout time.Duration `json:"timeout"`

	// SkipInitialStatus skips sending initial status.
	SkipInitialStatus bool `json:"skip_initial_status,omitempty"`

	// RetryPolicy defines reconnection behavior.
	RetryPolicy RetryPolicy `json:"retryPolicy,omitempty"`

	// HealthCheck defines health checking behavior.
	HealthCheck HealthCheckConfig `json:"healthCheck,omitempty"`

	// PluginParameters are passed to the external plugin.
	PluginParameters map[string]string `json:"pluginParameters,omitempty"`
}

// RetryPolicy defines how to handle connection failures.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of reconnection attempts.
	MaxAttempts int `json:"maxAttempts,omitempty"`

	// BackoffMultiplier for exponential backoff.
	BackoffMultiplier float64 `json:"backoffMultiplier,omitempty"`

	// MaxBackoff is the maximum backoff duration.
	MaxBackoff time.Duration `json:"maxBackoff,omitempty"`

	// InitialBackoff is the initial backoff duration.
	InitialBackoff time.Duration `json:"initialBackoff,omitempty"`
}

// HealthCheckConfig defines health checking parameters.
type HealthCheckConfig struct {
	// Interval between health checks.
	Interval time.Duration `json:"interval,omitempty"`

	// Timeout for health check calls.
	Timeout time.Duration `json:"timeout,omitempty"`

	// ErrorThreshold defines when to consider plugin unhealthy.
	ErrorThreshold int `json:"errorThreshold,omitempty"`
}

// ConditionDefinition defines a condition that the monitor can report.
type ConditionDefinition struct {
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// ApplyConfiguration applies default values and parses duration strings.
func (config *ExternalMonitorConfig) ApplyConfiguration() error {
	// Set default values
	if config.PluginConfig.InvokeInterval == 0 {
		config.PluginConfig.InvokeInterval = 30 * time.Second
	}
	if config.PluginConfig.Timeout == 0 {
		config.PluginConfig.Timeout = 10 * time.Second
	}

	// Set retry policy defaults
	if config.PluginConfig.RetryPolicy.MaxAttempts == 0 {
		config.PluginConfig.RetryPolicy.MaxAttempts = 5
	}
	if config.PluginConfig.RetryPolicy.BackoffMultiplier == 0 {
		config.PluginConfig.RetryPolicy.BackoffMultiplier = 2.0
	}
	if config.PluginConfig.RetryPolicy.MaxBackoff == 0 {
		config.PluginConfig.RetryPolicy.MaxBackoff = 5 * time.Minute
	}
	if config.PluginConfig.RetryPolicy.InitialBackoff == 0 {
		config.PluginConfig.RetryPolicy.InitialBackoff = 1 * time.Second
	}

	// Set health check defaults
	if config.PluginConfig.HealthCheck.Interval == 0 {
		config.PluginConfig.HealthCheck.Interval = 30 * time.Second
	}
	if config.PluginConfig.HealthCheck.Timeout == 0 {
		config.PluginConfig.HealthCheck.Timeout = 5 * time.Second
	}
	if config.PluginConfig.HealthCheck.ErrorThreshold == 0 {
		config.PluginConfig.HealthCheck.ErrorThreshold = 3
	}

	// Default metrics reporting to true
	if !config.MetricsReporting {
		config.MetricsReporting = true
	}

	return nil
}

// Validate checks the configuration for correctness.
func (config *ExternalMonitorConfig) Validate() error {
	if config.Plugin != "external" {
		return fmt.Errorf("plugin must be \"external\", got %q", config.Plugin)
	}

	if config.Source == "" {
		return fmt.Errorf("source is required")
	}

	if config.PluginConfig.SocketAddress == "" {
		return fmt.Errorf("socketAddress is required")
	}

	if config.PluginConfig.InvokeInterval < time.Second {
		return fmt.Errorf("invoke_interval must be at least 1 second")
	}

	if config.PluginConfig.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second")
	}

	if config.PluginConfig.Timeout >= config.PluginConfig.InvokeInterval {
		return fmt.Errorf("timeout must be less than invoke_interval")
	}

	// Validate retry policy
	if config.PluginConfig.RetryPolicy.MaxAttempts < 1 {
		return fmt.Errorf("retryPolicy.maxAttempts must be at least 1")
	}

	if config.PluginConfig.RetryPolicy.BackoffMultiplier < 1.0 {
		return fmt.Errorf("retryPolicy.backoffMultiplier must be at least 1.0")
	}

	// Validate health check
	if config.PluginConfig.HealthCheck.ErrorThreshold < 1 {
		return fmt.Errorf("healthCheck.errorThreshold must be at least 1")
	}

	// Validate conditions
	for i, condition := range config.Conditions {
		if condition.Type == "" {
			return fmt.Errorf("condition[%d].type is required", i)
		}
		if condition.Reason == "" {
			return fmt.Errorf("condition[%d].reason is required", i)
		}
		if condition.Message == "" {
			return fmt.Errorf("condition[%d].message is required", i)
		}
	}

	return nil
}
