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

package plugin

import (
	"testing"
	"time"

	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
)

func TestNewPluginRun(t *testing.T) {
	ruleTimeout := 1 * time.Second

	utMetas := map[string]struct {
		Rule       cpmtypes.CustomRule
		ExitStatus cpmtypes.Status
		Output     string
	}{
		"ok": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/ok.sh",
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.OK,
			Output:     "OK",
		},
		"non-ok": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/non-ok.sh",
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.NonOK,
			Output:     "NonOK",
		},
		"unknown": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/unknown.sh",
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.Unknown,
			Output:     "UNKNOWN",
		},
		"non executable": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/non-executable.sh",
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.Unknown,
			Output:     "Error in starting plugin. Please check the error log",
		},
		"longer than 80 stdout with ok exit status": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/longer-than-80-stdout-with-ok-exit-status.sh",
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.OK,
			Output:     "01234567890123456789012345678901234567890123456789012345678901234567890123456789",
		},
		"non defined exit status": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/non-defined-exit-status.sh",
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.Unknown,
			Output:     "NON-DEFINED-EXIT-STATUS",
		},
		"sleep 3 second with ok exit status": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/sleep-3-second-with-ok-exit-status.sh",
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.Unknown,
			Output:     `Timeout when running plugin "./test-data/sleep-3-second-with-ok-exit-status.sh": state - signal: killed. output - ""`,
		},
	}

	conf := cpmtypes.CustomPluginConfig{}
	(&conf).ApplyConfiguration()
	p := Plugin{config: conf}
	for desp, utMeta := range utMetas {
		gotExitStatus, gotOutput := p.run(utMeta.Rule)
		// cut at position max_output_length if expected output is longer than max_output_length bytes
		if len(utMeta.Output) > *p.config.PluginGlobalConfig.MaxOutputLength {
			utMeta.Output = utMeta.Output[:*p.config.PluginGlobalConfig.MaxOutputLength]
		}
		if gotExitStatus != utMeta.ExitStatus || gotOutput != utMeta.Output {
			t.Errorf("%s", desp)
			t.Errorf("Error in run plugin and get exit status and output for %q. "+
				"Got exit status: %v, Expected exit status: %v. "+
				"Got output: %q, Expected output: %q",
				utMeta.Rule.Path, gotExitStatus, utMeta.ExitStatus, gotOutput, utMeta.Output)
		}
	}
}
