/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package logcounter

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/clock"

	"k8s.io/node-problem-detector/pkg/logcounter/types"
	"k8s.io/node-problem-detector/pkg/systemlogmonitor"
	systemtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

func NewTestLogCounter(pattern string, startTime time.Time) (types.LogCounter, *clock.FakeClock, chan *systemtypes.Log) {
	logCh := make(chan *systemtypes.Log)
	clock := clock.NewFakeClock(startTime)
	return &logCounter{
		logCh:   logCh,
		buffer:  systemlogmonitor.NewLogBuffer(bufferSize),
		pattern: pattern,
		clock:   clock,
	}, clock, logCh
}

func TestCount(t *testing.T) {
	startTime := time.Now()
	for _, tc := range []struct {
		description   string
		logs          []*systemtypes.Log
		pattern       string
		expectedCount int
	}{
		{
			description:   "no logs",
			logs:          []*systemtypes.Log{},
			pattern:       "",
			expectedCount: 0,
		},
		{
			description: "one matching log",
			logs: []*systemtypes.Log{
				{
					Timestamp: startTime.Add(-time.Second),
					Message:   "0",
				},
			},
			pattern:       "0",
			expectedCount: 1,
		},
		{
			description: "one non-matching log",
			logs: []*systemtypes.Log{
				{
					Timestamp: startTime.Add(-time.Second),
					Message:   "1",
				},
			},
			pattern:       "0",
			expectedCount: 0,
		},
		{
			description: "log too new",
			logs: []*systemtypes.Log{
				{
					Timestamp: startTime.Add(time.Second),
					Message:   "0",
				},
			},
			pattern:       "0",
			expectedCount: 0,
		},
		{
			description: "many logs",
			logs: []*systemtypes.Log{
				{
					Timestamp: startTime.Add(-time.Second),
					Message:   "0",
				},
				{
					Timestamp: startTime.Add(-time.Second),
					Message:   "0",
				},
				{
					Timestamp: startTime.Add(-time.Second),
					Message:   "1",
				},
				{
					Timestamp: startTime.Add(time.Second),
					Message:   "0",
				},
			},
			pattern:       "0",
			expectedCount: 2,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			counter, fakeClock, logCh := NewTestLogCounter(tc.pattern, startTime)
			go func(logs []*systemtypes.Log, ch chan<- *systemtypes.Log) {
				for _, log := range logs {
					ch <- log
				}
				// trigger the timeout to ensure the test doesn't block permanently
				for {
					fakeClock.Step(2 * timeout)
				}
			}(tc.logs, logCh)
			actualCount, err := counter.Count()
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			if actualCount != tc.expectedCount {
				t.Errorf("got %d; expected %d", actualCount, tc.expectedCount)
			}
		})
	}
}
