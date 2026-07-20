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
	"reflect"
	"strings"
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
	messageChangeBasedConditionUpdate := true
	disableMetricsReporting := false
	disableInitialStatusUpdate := true

	ruleTimeout := 1 * time.Second
	ruleTimeoutString := ruleTimeout.String()
	ruleInvokeInterval := 7 * time.Second
	ruleInvokeIntervalString := ruleInvokeInterval.String()
	invalidRuleInvokeIntervalString := "invalid"

	utMetas := map[string]struct {
		Orig              CustomPluginConfig
		Wanted            CustomPluginConfig
		ErrorMessageStart string
	}{
		"global default settings": {
			Orig: CustomPluginConfig{
				Rules: []*CustomRule{
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
					InvokeIntervalString:                    &defaultInvokeIntervalString,
					InvokeInterval:                          &defaultInvokeInterval,
					TimeoutString:                           &defaultGlobalTimeoutString,
					Timeout:                                 &defaultGlobalTimeout,
					MaxOutputLength:                         &defaultMaxOutputLength,
					Concurrency:                             &defaultConcurrency,
					EnableMessageChangeBasedConditionUpdate: &defaultMessageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &defaultSkipInitialStatus,
				},
				EnableMetricsReporting: &defaultEnableMetricsReporting,
				Rules: []*CustomRule{
					{
						Path:                 "../plugin/test-data/ok.sh",
						InvokeIntervalString: nil,
						InvokeInterval:       nil,
					},
					{
						Path:                 "../plugin/test-data/warning.sh",
						Timeout:              &ruleTimeout,
						TimeoutString:        &ruleTimeoutString,
						InvokeIntervalString: nil,
						InvokeInterval:       nil,
					},
				},
			},
		},
		"custom rule invoke interval": {
			Orig: CustomPluginConfig{
				Rules: []*CustomRule{
					{
						Path:                 "../plugin/test-data/ok.sh",
						InvokeIntervalString: &ruleInvokeIntervalString,
					},
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString:                    &defaultInvokeIntervalString,
					InvokeInterval:                          &defaultInvokeInterval,
					TimeoutString:                           &defaultGlobalTimeoutString,
					Timeout:                                 &defaultGlobalTimeout,
					MaxOutputLength:                         &defaultMaxOutputLength,
					Concurrency:                             &defaultConcurrency,
					EnableMessageChangeBasedConditionUpdate: &defaultMessageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &defaultSkipInitialStatus,
				},
				EnableMetricsReporting: &defaultEnableMetricsReporting,
				Rules: []*CustomRule{
					{
						Path:                 "../plugin/test-data/ok.sh",
						InvokeIntervalString: &ruleInvokeIntervalString,
						InvokeInterval:       &ruleInvokeInterval,
					},
				},
			},
		},
		"invalid rule invoke interval": {
			Orig: CustomPluginConfig{
				Rules: []*CustomRule{
					{
						Path:                 "../plugin/test-data/ok.sh",
						InvokeIntervalString: &invalidRuleInvokeIntervalString,
					},
				},
			},
			ErrorMessageStart: "error in parsing rule invoke interval",
		},
		"custom invoke interval": {
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString: &invokeIntervalString,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString:                    &invokeIntervalString,
					InvokeInterval:                          &invokeInterval,
					TimeoutString:                           &defaultGlobalTimeoutString,
					Timeout:                                 &defaultGlobalTimeout,
					MaxOutputLength:                         &defaultMaxOutputLength,
					Concurrency:                             &defaultConcurrency,
					EnableMessageChangeBasedConditionUpdate: &defaultMessageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &defaultSkipInitialStatus,
				},
				EnableMetricsReporting: &defaultEnableMetricsReporting,
			},
		},
		"custom default timeout": {
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					TimeoutString: &globalTimeoutString,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString:                    &defaultInvokeIntervalString,
					InvokeInterval:                          &defaultInvokeInterval,
					TimeoutString:                           &globalTimeoutString,
					Timeout:                                 &globalTimeout,
					MaxOutputLength:                         &defaultMaxOutputLength,
					Concurrency:                             &defaultConcurrency,
					EnableMessageChangeBasedConditionUpdate: &defaultMessageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &defaultSkipInitialStatus,
				},
				EnableMetricsReporting: &defaultEnableMetricsReporting,
			},
		},
		"custom max output length": {
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					MaxOutputLength: &maxOutputLength,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString:                    &defaultInvokeIntervalString,
					InvokeInterval:                          &defaultInvokeInterval,
					TimeoutString:                           &defaultGlobalTimeoutString,
					Timeout:                                 &defaultGlobalTimeout,
					MaxOutputLength:                         &maxOutputLength,
					Concurrency:                             &defaultConcurrency,
					EnableMessageChangeBasedConditionUpdate: &defaultMessageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &defaultSkipInitialStatus,
				},
				EnableMetricsReporting: &defaultEnableMetricsReporting,
			},
		},
		"custom concurrency": {
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					Concurrency: &concurrency,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString:                    &defaultInvokeIntervalString,
					InvokeInterval:                          &defaultInvokeInterval,
					TimeoutString:                           &defaultGlobalTimeoutString,
					Timeout:                                 &defaultGlobalTimeout,
					MaxOutputLength:                         &defaultMaxOutputLength,
					Concurrency:                             &concurrency,
					EnableMessageChangeBasedConditionUpdate: &defaultMessageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &defaultSkipInitialStatus,
				},
				EnableMetricsReporting: &defaultEnableMetricsReporting,
			},
		},
		"custom message change based condition update": {
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					EnableMessageChangeBasedConditionUpdate: &messageChangeBasedConditionUpdate,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString:                    &defaultInvokeIntervalString,
					InvokeInterval:                          &defaultInvokeInterval,
					TimeoutString:                           &defaultGlobalTimeoutString,
					Timeout:                                 &defaultGlobalTimeout,
					MaxOutputLength:                         &defaultMaxOutputLength,
					Concurrency:                             &defaultConcurrency,
					EnableMessageChangeBasedConditionUpdate: &messageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &defaultSkipInitialStatus,
				},
				EnableMetricsReporting: &defaultEnableMetricsReporting,
			},
		},
		"disable metrics reporting": {
			Orig: CustomPluginConfig{
				EnableMetricsReporting: &disableMetricsReporting,
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString:                    &defaultInvokeIntervalString,
					InvokeInterval:                          &defaultInvokeInterval,
					TimeoutString:                           &defaultGlobalTimeoutString,
					Timeout:                                 &defaultGlobalTimeout,
					MaxOutputLength:                         &defaultMaxOutputLength,
					Concurrency:                             &defaultConcurrency,
					EnableMessageChangeBasedConditionUpdate: &defaultMessageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &defaultSkipInitialStatus,
				},
				EnableMetricsReporting: &disableMetricsReporting,
			},
		},
		"disable status update during initialization": {
			Orig: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					SkipInitialStatus: &disableInitialStatusUpdate,
				},
			},
			Wanted: CustomPluginConfig{
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeIntervalString:                    &defaultInvokeIntervalString,
					InvokeInterval:                          &defaultInvokeInterval,
					TimeoutString:                           &defaultGlobalTimeoutString,
					Timeout:                                 &defaultGlobalTimeout,
					MaxOutputLength:                         &defaultMaxOutputLength,
					Concurrency:                             &defaultConcurrency,
					EnableMessageChangeBasedConditionUpdate: &defaultMessageChangeBasedConditionUpdate,
					SkipInitialStatus:                       &disableInitialStatusUpdate,
				},
				EnableMetricsReporting: &defaultEnableMetricsReporting,
			},
		},
	}

	for desp, utMeta := range utMetas {
		err := (&utMeta.Orig).ApplyConfiguration()
		if utMeta.ErrorMessageStart != "" {
			if err == nil {
				t.Errorf("Error in apply configuration for %q: wanted an error got nil", desp)
				continue
			}
			if !strings.HasPrefix(err.Error(), utMeta.ErrorMessageStart) {
				t.Errorf("Error in apply configuration for %q: wanted prefix %q, got %q", desp, utMeta.ErrorMessageStart, err)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("%+v", utMeta.Orig.Rules[0])) {
				t.Errorf("Error in apply configuration for %q does not include rule %+v: %v", desp, utMeta.Orig.Rules[0], err)
			}
			continue
		}
		if err != nil {
			t.Errorf("Error in apply configuration for %q: %v", desp, err)
		}
		if !reflect.DeepEqual(utMeta.Orig, utMeta.Wanted) {
			t.Errorf("Error in apply configuration for %q", desp)
			t.Errorf("Wanted: %+v. \nGot: %+v", utMeta.Wanted, utMeta.Orig)
		}
	}
}

func TestCustomPluginConfigValidate(t *testing.T) {
	normalRuleTimeout := defaultGlobalTimeout - 1*time.Second
	exceededRuleTimeout := defaultGlobalTimeout + 1*time.Second
	zeroInvokeInterval := time.Duration(0)
	negativeInvokeInterval := -1 * time.Second

	utMetas := map[string]struct {
		Conf              CustomPluginConfig
		IsError           bool
		ErrorContains     string
		ErrorIncludesRule bool
	}{
		"normal": {
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*CustomRule{
					{
						Path:    "../plugin/test-data/ok.sh",
						Timeout: &normalRuleTimeout,
					},
				},
			},
			IsError: false,
		},
		"non exist plugin path": {
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*CustomRule{
					{
						Path:    "../plugin/test-data/non-exist-plugin-path.sh",
						Timeout: &normalRuleTimeout,
					},
				},
			},
			IsError: true,
		},
		"non supported plugin": {
			// non supported plugin
			Conf: CustomPluginConfig{
				Plugin: "non-supported-plugin",
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*CustomRule{
					{
						Path:    "../plugin/test-data/non-exist-plugin-path.sh",
						Timeout: &normalRuleTimeout,
					},
				},
			},
			IsError: true,
		},
		"exceed global timeout": {
			// exceed global timeout
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*CustomRule{
					{
						Path:    "../plugin/test-data/ok.sh",
						Timeout: &exceededRuleTimeout,
					},
				},
			},
			IsError: true,
		},
		"zero rule invoke interval": {
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*CustomRule{
					{
						Path:           "../plugin/test-data/ok.sh",
						InvokeInterval: &zeroInvokeInterval,
					},
				},
			},
			IsError:           true,
			ErrorContains:     "Rule:",
			ErrorIncludesRule: true,
		},
		"negative rule invoke interval": {
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*CustomRule{
					{
						Path:           "../plugin/test-data/ok.sh",
						InvokeInterval: &negativeInvokeInterval,
					},
				},
			},
			IsError:           true,
			ErrorContains:     "Rule:",
			ErrorIncludesRule: true,
		},
		"zero global invoke interval": {
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &zeroInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
			},
			IsError:       true,
			ErrorContains: "global invoke interval",
		},
		"permanent problem has preset default condition": {
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				DefaultConditions: []types.Condition{
					{
						Type:    "TestCondition",
						Reason:  "TestConditionOK",
						Message: "Test condition is OK.",
					},
				},
				Rules: []*CustomRule{
					{
						Type:      types.Perm,
						Condition: "TestCondition",
						Reason:    "TestConditionFail",
						Path:      "../plugin/test-data/ok.sh",
						Timeout:   &normalRuleTimeout,
					},
				},
			},
			IsError: false,
		},
		"permanent problem does not have preset default condition": {
			Conf: CustomPluginConfig{
				Plugin: customPluginName,
				PluginGlobalConfig: pluginGlobalConfig{
					InvokeInterval:  &defaultInvokeInterval,
					Timeout:         &defaultGlobalTimeout,
					MaxOutputLength: &defaultMaxOutputLength,
					Concurrency:     &defaultConcurrency,
				},
				Rules: []*CustomRule{
					{
						Type:      types.Perm,
						Condition: "TestCondition",
						Reason:    "TestConditionFail",
						Path:      "../plugin/test-data/ok.sh",
						Timeout:   &normalRuleTimeout,
					},
				},
			},
			IsError: true,
		},
	}

	for desp, utMeta := range utMetas {
		err := utMeta.Conf.Validate()
		if err != nil && !utMeta.IsError {
			t.Error(desp)
			t.Errorf("Error in validating custom plugin configuration %+v. Wanted nil got an error", utMeta)
		}
		if err == nil && utMeta.IsError {
			t.Error(desp)
			t.Errorf("Error in validating custom plugin configuration %+v. Wanted an error got nil", utMeta)
		}
		if err != nil && utMeta.ErrorContains != "" && !strings.Contains(err.Error(), utMeta.ErrorContains) {
			t.Errorf("Error in validating %q: wanted error containing %q, got %q", desp, utMeta.ErrorContains, err)
		}
		if err != nil && utMeta.ErrorIncludesRule && !strings.Contains(err.Error(), fmt.Sprintf("%+v", utMeta.Conf.Rules[0])) {
			t.Errorf("Error in validating %q does not include rule %+v: %v", desp, utMeta.Conf.Rules[0], err)
		}
	}
}
