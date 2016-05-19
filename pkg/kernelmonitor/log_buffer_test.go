/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package kernelmonitor

import (
	"reflect"
	"testing"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/types"
)

func TestPush(t *testing.T) {
	for c, test := range []struct {
		max      int
		logs     []string
		expected string
	}{
		{
			max:      1,
			logs:     []string{"a", "b"},
			expected: "b",
		},
		{
			max:      2,
			logs:     []string{"a", "b"},
			expected: "a\nb",
		},
		{
			max:      2,
			logs:     []string{"a", "b", "c"},
			expected: "b\nc",
		},
		{
			max:      2,
			logs:     []string{"a", "b", "c", "d"},
			expected: "c\nd",
		},
	} {
		b := NewLogBuffer(test.max)
		for _, log := range test.logs {
			b.Push(&types.KernelLog{Message: log})
		}
		got := b.String()
		if test.expected != got {
			t.Errorf("case %d: expected %q, got %q", c+1, test.expected, got)
		}
	}
}

func TestMatch(t *testing.T) {
	max := 4
	for c, test := range []struct {
		logs     []string
		exprs    []string
		expected [][]string
	}{
		{
			// Buffer not full
			logs: []string{"a1", "b2"},
			exprs: []string{
				"a1",     // Not including the last line, should not match
				"b1",     // Not match
				"b2",     // match
				`\w{2}`,  // Regexp should work
				"a1\nb2", // Including the last line, should match
				`a1b2`,   // No new line, should not match
			},
			expected: [][]string{{}, {}, {"b2"}, {"b2"}, {"a1", "b2"}, {}},
		},
		{
			// Buffer full
			logs: []string{"a1", "b2", "c3", "d4", "e5"},
			exprs: []string{
				"(?s)a1.+",         // Rotate out, should not match
				`[a-z]\d\n[a-z]\d`, // New line should work, and only the one contains the last line should match
				`[a-z]\d`,          // Multiple match, only the one contains the last line should match
			},
			expected: [][]string{{}, {"d4", "e5"}, {"e5"}},
		},
	} {
		b := NewLogBuffer(max)
		for _, log := range test.logs {
			b.Push(&types.KernelLog{Message: log})
		}
		for i, expr := range test.exprs {
			kLogs := b.Match(expr)
			got := []string{}
			for _, kLog := range kLogs {
				got = append(got, kLog.Message)
			}
			if !reflect.DeepEqual(test.expected[i], got) {
				t.Errorf("case %d.%d: expected %v, got %v", c+1, i+1, test.expected[i], got)
			}
		}
	}
}
