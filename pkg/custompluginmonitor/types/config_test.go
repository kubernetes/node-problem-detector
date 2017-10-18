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
	"reflect"
	"testing"
	"time"

	"k8s.io/node-problem-detector/pkg/types"
)

func TestCustomPluginConfigApplyConfiguration(t *testing.T) {
	globalTimeout := 6 * time.Second
	globalTimeoutString := globalTimeout.String()
	invokeInterval := 31 * time.Second
	invokeIntervalString := invokeInterval.String()
	maxOutputLength := 79
	concurrency := 2

	ruleTimeout := 1 * time.Second
	ruleTimeoutString := ruleTimeout.String()

	utMetas := []struct {
		Orig   CustomPluginConfig
		Wanted CustomPluginConfig
	}{
		{
			Orig: CustomPluginConfig{
				Rules: []*types.CustomRule{
					{
						Path: "../plugin/test-data/ok.sh",
					},
					{
						Path:          "../plugin/test-data/warning.sh",
						TimeoutString: &ruleTimeoutString,
					},
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString: &defaultInvokeIntervalString,
					InvokeInterval:       &defaultInvokeInterval,
					TimeoutString:        &defaultGlobalTimeoutString,
					Timeout:              &defaultGlobalTimeout,
					MaxOutputLength:      &defaultMaxOutputLength,
					Concurrency:          &defaultConcurrency,
				},
				Rules: []*types.CustomRule{
					{
						Path: "../plugin/test-data/ok.sh",
					},
					{
						Path:          "../plugin/test-data/warning.sh",
						Timeout:       &ruleTimeout,
						TimeoutString: &ruleTimeoutString,
					},
				},
			},
		},
		{
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString: &invokeIntervalString,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString: &invokeIntervalString,
					InvokeInterval:       &invokeInterval,
					TimeoutString:        &defaultGlobalTimeoutString,
					Timeout:              &defaultGlobalTimeout,
					MaxOutputLength:      &defaultMaxOutputLength,
					Concurrency:          &defaultConcurrency,
				},
			},
		},
		{
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					TimeoutString: &globalTimeoutString,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString: &defaultInvokeIntervalString,
					InvokeInterval:       &defaultInvokeInterval,
					TimeoutString:        &globalTimeoutString,
					Timeout:              &globalTimeout,
					MaxOutputLength:      &defaultMaxOutputLength,
					Concurrency:          &defaultConcurrency,
				},
			},
		},
		{
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					MaxOutputLength: &maxOutputLength,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString: &defaultInvokeIntervalString,
					InvokeInterval:       &defaultInvokeInterval,
					TimeoutString:        &defaultGlobalTimeoutString,
					Timeout:              &defaultGlobalTimeout,
					MaxOutputLength:      &maxOutputLength,
					Concurrency:          &defaultConcurrency,
				},
			},
		},
		{
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					Concurrency: &concurrency,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString: &defaultInvokeIntervalString,
					InvokeInterval:       &defaultInvokeInterval,
					TimeoutString:        &defaultGlobalTimeoutString,
					Timeout:              &defaultGlobalTimeout,
					MaxOutputLength:      &defaultMaxOutputLength,
					Concurrency:          &concurrency,
				},
			},
		},
	}

	for _, utMeta := range utMetas {
		(&utMeta.Orig).ApplyConfiguration()
		if !reflect.DeepEqual(utMeta.Orig, utMeta.Wanted) {
			t.Errorf("Error in apply configuration.")
			t.Errorf("Wanted: %+v. \nGot: %+v", utMeta.Wanted, utMeta.Orig)
		}
	}
}

func TestCustomPluginConfigValidate(t *testing.T) {
	normalRuleTimeout := defaultGlobalTimeout - 1*time.Second
	exceededRuleTimeout := defaultGlobalTimeout + 1*time.Second

	utMetas := []struct {
		Conf    CustomPluginConfig
		IsError bool
	}{
		{
			// normal
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*types.CustomRule{
					{
						Path:    "../plugin/test-data/ok.sh",
						Timeout: &normalRuleTimeout,
					},
				},
			},
			IsError: false,
		},
		{
			// non exist plugin path
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*types.CustomRule{
					{
						Path:    "../plugin/test-data/non-exist-plugin-path.sh",
						Timeout: &normalRuleTimeout,
					},
				},
			},
			IsError: true,
		},
		{
			// non supported plugin
			Conf: CustomPluginConfig{
				Plugin: "non-supported-plugin",
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*types.CustomRule{
					{
						Path:    "../plugin/test-data/non-exist-plugin-path.sh",
						Timeout: &normalRuleTimeout,
					},
				},
			},
			IsError: true,
		},
		{
			// exceed global timeout
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*types.CustomRule{
					{
						Path:    "../plugin/test-data/ok.sh",
						Timeout: &exceededRuleTimeout,
					},
				},
			},
			IsError: true,
		},
	}

	for _, utMeta := range utMetas {
		err := utMeta.Conf.Validate()
		if err != nil && !utMeta.IsError {
			t.Errorf("Error in validating custom plugin configuration %+v. Want an error got nil", utMeta)
		}
		if err == nil && utMeta.IsError {
			t.Errorf("Error in validating custom plugin configuration %+v. Want nil got an error", utMeta)
		}
	}
}
