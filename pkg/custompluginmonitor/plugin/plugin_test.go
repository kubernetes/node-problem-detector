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
	"runtime"
	"strings"
	"testing"
	"time"

	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
)

func TestNewPluginRun(t *testing.T) {
	ruleTimeout := 1 * time.Second
	timeoutExitStatus := cpmtypes.Unknown
	ext := "sh"

	if runtime.GOOS == "windows" {
		ext = "cmd"
		timeoutExitStatus = cpmtypes.NonOK
	}

	utMetas := map[string]struct {
		Rule       cpmtypes.CustomRule
		ExitStatus cpmtypes.Status
		Output     string
	}{
		"ok": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/ok." + ext,
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.OK,
			Output:     "OK",
		},
		"non-ok": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/non-ok." + ext,
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.NonOK,
			Output:     "NonOK",
		},
		"unknown": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/unknown." + ext,
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.Unknown,
			Output:     "UNKNOWN",
		},
		"non executable": {
			Rule: cpmtypes.CustomRule{
				// Intentionally run .sh for Windows, this is meant to be not executable.
				Path:    "./test-data/non-executable.sh",
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.Unknown,
			Output:     "Error in starting plugin. Please check the error log",
		},
		"longer than 80 stdout with ok exit status": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/longer-than-80-stdout-with-ok-exit-status." + ext,
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.OK,
			Output:     "01234567890123456789012345678901234567890123456789012345678901234567890123456789",
		},
		"non defined exit status": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/non-defined-exit-status." + ext,
				Timeout: &ruleTimeout,
			},
			ExitStatus: cpmtypes.Unknown,
			Output:     "NON-DEFINED-EXIT-STATUS",
		},
		"sleep 3 second with ok exit status": {
			Rule: cpmtypes.CustomRule{
				Path:    "./test-data/sleep-3-second-with-ok-exit-status." + ext,
				Timeout: &ruleTimeout,
			},
			ExitStatus: timeoutExitStatus,
			Output:     `Timeout when running plugin "./test-data/sleep-3-second-with-ok-exit-status.` + ext + `": state - signal: killed. output - ""`,
		},
	}

	for k, v := range utMetas {
		desp := k
		utMeta := v
		t.Run(desp, func(t *testing.T) {
			conf := cpmtypes.CustomPluginConfig{}
			if err := (&conf).ApplyConfiguration(); err != nil {
				t.Errorf("Failed to apply configuration: %v", err)
			}
			p := Plugin{config: conf}
			gotExitStatus, gotOutput := p.run(utMeta.Rule)
			// cut at position max_output_length if expected output is longer than max_output_length bytes
			if len(utMeta.Output) > *p.config.PluginGlobalConfig.MaxOutputLength {
				utMeta.Output = utMeta.Output[:*p.config.PluginGlobalConfig.MaxOutputLength]
			}
			if gotExitStatus != utMeta.ExitStatus || gotOutput != utMeta.Output {
				t.Errorf("Error in run plugin and get exit status and output for %q. "+
					"Got exit status: %v, Expected exit status: %v. "+
					"Got output: %q, Expected output: %q",
					utMeta.Rule.Path, gotExitStatus, utMeta.ExitStatus, gotOutput, utMeta.Output)
			}
		})
	}
}

// TestPluginRunCaptureRespectsMaxOutputLength verifies that a plugin emitting
// more than the old hardcoded 4 KiB capture buffer is captured up to the
// configured max_output_length, not silently truncated at 4096 bytes.
func TestPluginRunCaptureRespectsMaxOutputLength(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture is a shell script; this path is exercised on non-Windows platforms")
	}

	ruleTimeout := 1 * time.Second
	// Larger than the old 4096-byte capture buffer, and smaller than the
	// 16384 bytes the fixture emits, so the result must be exactly this many
	// bytes if (and only if) capture honors max_output_length.
	maxOutputLength := 8192

	conf := cpmtypes.CustomPluginConfig{}
	// Set before ApplyConfiguration so it is not overwritten by the default,
	// and use a fresh pointer so we don't mutate the package-level default.
	conf.PluginGlobalConfig.MaxOutputLength = &maxOutputLength
	if err := (&conf).ApplyConfiguration(); err != nil {
		t.Fatalf("Failed to apply configuration: %v", err)
	}
	p := Plugin{config: conf}

	rule := cpmtypes.CustomRule{
		Path:    "./test-data/large-stdout-with-ok-exit-status.sh",
		Timeout: &ruleTimeout,
	}
	gotStatus, gotOutput := p.run(rule)

	if gotStatus != cpmtypes.OK {
		t.Errorf("exit status: got %v, want %v", gotStatus, cpmtypes.OK)
	}
	if len(gotOutput) != maxOutputLength {
		t.Errorf("output length: got %d, want %d (must be capped at max_output_length, not a fixed 4 KiB buffer)",
			len(gotOutput), maxOutputLength)
	}
	if want := strings.Repeat("a", maxOutputLength); gotOutput != want {
		t.Errorf("output content mismatch: got %d bytes, want %d 'a' bytes", len(gotOutput), len(want))
	}
}
