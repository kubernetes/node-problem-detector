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
	"math/rand"
	"reflect"
	"regexp"
	"slices"
	"testing"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

// referenceMatch is the unoptimized matcher: build the whole buffer, scan all of it.
// Match must agree with it on every input.
func referenceMatch(b *logBuffer, reg *regexp.Regexp) []*types.Log {
	log := concatLogs(append(append([]string{}, b.msg[b.current%b.max:]...), b.msg[:b.current%b.max]...))
	loc := reg.FindStringIndex(log)
	if loc == nil {
		return nil
	}
	s := len(log) - loc[0] - 1
	total := 0
	matched := []*types.Log{}
	for i := b.tail(); i >= b.current && b.buffer[i%b.max] != nil; i-- {
		matched = append(matched, b.buffer[i%b.max])
		total += len(b.msg[i%b.max]) + 1
		if total > s {
			break
		}
	}
	slices.Reverse(matched)
	return matched
}

// equivalencePatterns mixes shipped rules with rules that attack the last-line shortcut.
var equivalencePatterns = []string{
	// Shipped rules.
	`Killed process \d+ (.+) total-vm:\d+kB, anon-rss:\d+kB, file-rss:\d+kB.*`,
	`task [\S ]+:\w+ blocked for more than \w+ seconds\.`,
	`unregister_netdevice: waiting for \w+ to become free. Usage count = \d+`,
	`BUG: unable to handle kernel NULL pointer dereference at .*`,
	`EXT4-fs error .*`,
	`XFS .* Shutting down filesystem.?`,
	`.*\[Hardware Error\]: event severity: fatal$`,
	`Error syncing pod .*skipping.*failed to "StartContainer".*`,
	// Rules that must fall back to the full buffer.
	`(?s)first.*second`,
	`^only line`,
	`(?m)^line \w+$`,
	`alpha\nbeta`,
	`alpha[\s\S]*beta`,
	`alpha[\x00-\x7f]+beta`,
	`\Aalpha`,
	// Top-level alternations must also stay on the last line.
	`alpha|beta`,
	`abort|abandon`,
	`alpha|`,
	// Rules that stay on the last line but stress the edges.
	`\balpha\b`,
	`alpha$`,
	`a*`,
	``,
	`(alpha|beta) gamma`,
	`[^x]+`,
}

// equivalenceLines are pushed in random order so matches land at every ring offset.
var equivalenceLines = []string{
	"alpha gamma",
	"beta gamma",
	"abort now",
	"only line",
	"first",
	"second",
	"line one",
	"line two",
	"alpha",
	"beta",
	"task docker:1234 blocked for more than 120 seconds.",
	"EXT4-fs error (device sda1): ext4_find_entry:1455: inode #2",
	"mce: [Hardware Error]: event severity: fatal",
	"Killed process 1234 (mysqld) total-vm:100kB, anon-rss:20kB, file-rss:3kB",
	"",
	"trailing\nembedded newline",
}

func TestMatchEquivalence(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	// Compile each rule once, before the trial loops.
	patterns := make([]*Pattern, 0, len(equivalencePatterns))
	refRegexps := make([]*regexp.Regexp, 0, len(equivalencePatterns))
	for _, expr := range equivalencePatterns {
		p, err := CompilePattern(expr)
		if err != nil {
			t.Fatalf("failed to compile %q: %v", expr, err)
		}
		patterns = append(patterns, p)
		refRegexps = append(refRegexps, regexp.MustCompile(`(?:`+expr+`)\z`))
	}
	for _, maxLines := range []int{1, 2, 3, 5, 10} {
		for trial := range 200 {
			buf := NewLogBuffer(maxLines)
			ref := NewLogBuffer(maxLines)
			// Push a random number of lines, from none to more than the ring.
			for range rng.Intn(maxLines*2 + 1) {
				log := &types.Log{Message: equivalenceLines[rng.Intn(len(equivalenceLines))]}
				buf.Push(log)
				ref.Push(log)
				for i, expr := range equivalencePatterns {
					want := referenceMatch(ref, refRegexps[i])
					got := buf.Match(patterns[i])
					if len(want) == 0 && len(got) == 0 {
						continue
					}
					if !reflect.DeepEqual(want, got) {
						t.Fatalf("maxLines=%d trial=%d pattern=%q buffer=%q:\nwant %v\ngot  %v",
							maxLines, trial, expr, ref.String(), messages(want), messages(got))
					}
				}
			}
		}
	}
}

func messages(logs []*types.Log) []string {
	out := []string{}
	for _, log := range logs {
		out = append(out, log.Message)
	}
	return out
}

// TestLastLineOnlyClassification pins the lastLineOnly verdict for each rule shape.
func TestLastLineOnlyClassification(t *testing.T) {
	for expr, want := range map[string]bool{
		`EXT4-fs error .*`: true,
		`task \S+ blocked`: true,
		`alpha$`:           true,
		`\balpha\b`:        true,
		// A negated class holds the newline unless the rule excludes it.
		`[^x]+`:                 false,
		`[^x\n]+`:               true,
		`(alpha|beta)+ gamma`:   true,
		`(?s)alpha.*beta`:       false,
		`^alpha`:                false,
		`\Aalpha`:               false,
		`(?m)^alpha$`:           false,
		"alpha\nbeta":           false,
		`alpha[\s\S]*beta`:      false,
		`alpha[\x00-\x7f]beta`:  false,
		`alpha[\n]beta`:         false,
		`alpha(beta|\n)`:        false,
		`alpha{1,3}[\t-\r]beta`: false,
		`alpha|beta`:            true,
		`abort|abandon`:         true,
		`a|`:                    true,
		`(alpha|beta) gamma`:    true,
	} {
		p, err := CompilePattern(expr)
		if err != nil {
			t.Fatalf("failed to compile %q: %v", expr, err)
		}
		if got := p.lastLineOnly; got != want {
			t.Errorf("pattern %q: lastLineOnly = %v, want %v", expr, got, want)
		}
	}
	// Every shipped kernel rule must keep the last line shortcut.
	for _, expr := range kernelMonitorPatterns {
		p, err := CompilePattern(expr)
		if err != nil {
			t.Fatalf("failed to compile %q: %v", expr, err)
		}
		if !p.lastLineOnly {
			t.Errorf("kernel rule %q: lastLineOnly = false, want true", expr)
		}
	}
}

// TestMatchAlternationAnchorsEveryBranch verifies that an earlier branch cannot
// match a stale buffered line.
func TestMatchAlternationAnchorsEveryBranch(t *testing.T) {
	b := NewLogBuffer(2)
	b.Push(&types.Log{Message: "kernel: oom-kill:constraint=CONSTRAINT_NONE"})
	b.Push(&types.Log{Message: "kubelet: node ready"})
	expr := `oom-kill|Out of memory`
	p, err := CompilePattern(expr)
	if err != nil {
		t.Fatalf("failed to compile %q: %v", expr, err)
	}
	if got := b.Match(p); len(got) != 0 {
		t.Fatalf("pattern %q matched stale logs: %v", expr, messages(got))
	}

	last := &types.Log{Message: "kernel: Out of memory"}
	b.Push(last)
	got := b.Match(p)
	want := []*types.Log{last}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("pattern %q: want %v, got %v", expr, messages(want), messages(got))
	}
}
