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

package makers

import (
	"regexp"
	"strings"
	"testing"
)

// Regexes are copied from the configs the e2e NPD loads (kernel-monitor.json,
// readonly-monitor.json); the injected lines must keep matching them or the
// e2e assertions in test/e2e/metriconly/metrics_test.go would fail on a real VM.
func TestExt4FilesystemErrorMatchesMonitorRegexes(t *testing.T) {
	lines := strings.Split(ext4ErrorPattern, "\n")
	cases := []struct {
		name  string
		regex string
	}{
		{"Ext4Error counter", `EXT4-fs error .*`},
		{"ReadonlyFilesystem gauge", `Remounting filesystem read-only`},
	}
	for _, tc := range cases {
		re := regexp.MustCompile(tc.regex)
		matched := false
		for _, line := range lines {
			if re.MatchString(line) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("no injected line matches %s regex %q; injected lines = %q", tc.name, tc.regex, lines)
		}
	}
}
