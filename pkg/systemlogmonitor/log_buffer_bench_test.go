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

package systemlogmonitor

import (
	"fmt"
	"testing"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

// kernelMonitorPatterns is the rule set of config/kernel-monitor.json.
var kernelMonitorPatterns = []string{
	`Killed process \d+ (.+) total-vm:\d+kB, anon-rss:\d+kB, file-rss:\d+kB.*`,
	`task [\S ]+:\w+ blocked for more than \w+ seconds\.`,
	`unregister_netdevice: waiting for \w+ to become free. Usage count = \d+`,
	`BUG: unable to handle kernel NULL pointer dereference at .*`,
	`divide error: 0000 \[#\d+\] SMP`,
	`EXT4-fs error .*`,
	`EXT4-fs warning .*`,
	`Buffer I/O error .*`,
	`XFS .* Shutting down filesystem.?`,
	`CE memory read error .*`,
	`.*\[Hardware Error\]: event severity: corrected$`,
	`.*\[Hardware Error\]: event severity: recoverable$`,
	`.*\[Hardware Error\]: event severity: fatal$`,
	`task docker:\w+ blocked for more than \w+ seconds\.`,
}

// benchmarkLine matches none of the rules above, the common case on a healthy node.
const benchmarkLine = "systemd[1]: Started Session 4321 of user core."

// BenchmarkPushAndMatchAll measures the per-line cost of the monitor hot path.
// Each iteration pushes one line and evaluates every rule against the buffer.
func BenchmarkPushAndMatchAll(b *testing.B) {
	for _, bufferSize := range []int{10, 100} {
		b.Run(fmt.Sprintf("buffer=%d", bufferSize), func(b *testing.B) {
			buf := NewLogBuffer(bufferSize)
			for range bufferSize {
				buf.Push(&types.Log{Message: benchmarkLine})
			}
			patterns := make([]*Pattern, 0, len(kernelMonitorPatterns))
			for _, expr := range kernelMonitorPatterns {
				p, err := CompilePattern(expr)
				if err != nil {
					b.Fatalf("failed to compile %q: %v", expr, err)
				}
				patterns = append(patterns, p)
			}
			log := &types.Log{Message: benchmarkLine}
			b.ReportAllocs()
			for b.Loop() {
				buf.Push(log)
				for _, p := range patterns {
					buf.Match(p)
				}
			}
		})
	}
}
