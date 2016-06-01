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
	"time"

	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/types"
)

const (
	testSource     = "TestSource"
	testConditionA = "TestConditionA"
	testConditionB = "TestConditionB"
)

func TestGenerateStatus(t *testing.T) {
	uptime := time.Unix(1000, 0)
	initConditions := []types.Condition{
		{
			Type:       testConditionA,
			Status:     true,
			Transition: time.Now(),
		},
		{
			Type:       testConditionB,
			Status:     false,
			Transition: time.Now(),
		},
	}
	logs := []*kerntypes.KernelLog{
		{
			Timestamp: 100000,
			Message:   "test message 1",
		},
		{
			Timestamp: 200000,
			Message:   "test message 2",
		},
	}
	for c, test := range []struct {
		rule     kerntypes.Rule
		expected types.Status
	}{
		// Do not need Pattern because we don't do pattern match in this test
		{
			rule: kerntypes.Rule{
				Type:      kerntypes.Perm,
				Condition: testConditionA,
				Reason:    "test reason",
			},
			expected: types.Status{
				Source: testSource,
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     true,
						Transition: time.Unix(1000, 100000*1000),
						Reason:     "test reason",
						Message:    "test message 1\ntest message 2",
					},
					initConditions[1],
				},
			},
		},
		{
			rule: kerntypes.Rule{
				Type:   kerntypes.Temp,
				Reason: "test reason",
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{{
					Severity:  types.Warn,
					Timestamp: time.Unix(1000, 100000*1000),
					Reason:    "test reason",
					Message:   "test message 1\ntest message 2",
				}},
				Conditions: initConditions,
			},
		},
	} {
		k := &kernelMonitor{
			config: MonitorConfig{
				Source: testSource,
			},
			conditions: initConditions,
			uptime:     uptime,
		}
		got := k.generateStatus(logs, test.rule)
		if !reflect.DeepEqual(&test.expected, got) {
			t.Errorf("case %d: expected status %+v, got %+v", c+1, test.expected, got)
		}
	}
}
