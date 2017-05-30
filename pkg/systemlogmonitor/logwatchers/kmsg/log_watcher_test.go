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

package kmsg

import (
	"testing"

	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/euank/go-kmsg-parser/kmsgparser"
	"github.com/stretchr/testify/assert"

	"time"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

type mockKmsgParser struct {
	kmsgs []kmsgparser.Message
}

func (m *mockKmsgParser) SetLogger(kmsgparser.Logger) {}
func (m *mockKmsgParser) Close() error                { return nil }
func (m *mockKmsgParser) Parse() <-chan kmsgparser.Message {
	c := make(chan kmsgparser.Message)
	go func() {
		for _, msg := range m.kmsgs {
			c <- msg
		}
	}()
	return c
}
func (m *mockKmsgParser) SeekEnd() error { return nil }

func TestWatch(t *testing.T) {
	now := time.Date(time.Now().Year(), time.January, 2, 3, 4, 5, 0, time.Local)
	fakeClock := fakeclock.NewFakeClock(now)
	testCases := []struct {
		log      *mockKmsgParser
		logs     []logtypes.Log
		lookback string
	}{
		{
			// The start point is at the head of the log file.
			log: &mockKmsgParser{kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(0 * time.Second)},
				{Message: "2", Timestamp: now.Add(1 * time.Second)},
				{Message: "3", Timestamp: now.Add(2 * time.Second)},
			}},
			logs: []logtypes.Log{
				{
					Timestamp: now,
					Message:   "1",
				},
				{
					Timestamp: now.Add(time.Second),
					Message:   "2",
				},
				{
					Timestamp: now.Add(2 * time.Second),
					Message:   "3",
				},
			},
			lookback: "0",
		},
		{
			// The start point is in the middle of the log file.
			log: &mockKmsgParser{kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(-1 * time.Second)},
				{Message: "2", Timestamp: now.Add(0 * time.Second)},
				{Message: "3", Timestamp: now.Add(1 * time.Second)},
			}},
			logs: []logtypes.Log{
				{
					Timestamp: now,
					Message:   "2",
				},
				{
					Timestamp: now.Add(time.Second),
					Message:   "3",
				},
			},
			lookback: "0",
		},
		{
			// The start point is at the end of the log file, but we look back.
			log: &mockKmsgParser{kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(-2 * time.Second)},
				{Message: "2", Timestamp: now.Add(-1 * time.Second)},
				{Message: "3", Timestamp: now.Add(0 * time.Second)},
			}},
			lookback: "1s",
			logs: []logtypes.Log{
				{
					Timestamp: now.Add(-time.Second),
					Message:   "2",
				},
				{
					Timestamp: now,
					Message:   "3",
				},
			},
		},
	}
	for _, test := range testCases {
		w := NewKmsgWatcher(types.WatcherConfig{Lookback: test.lookback})
		w.(*kernelLogWatcher).clock = fakeClock
		w.(*kernelLogWatcher).kmsgParser = test.log
		logCh, err := w.Watch()
		if err != nil {
			t.Fatal(err)
		}
		defer w.Stop()
		for _, expected := range test.logs {
			got := <-logCh
			assert.Equal(t, &expected, got)
		}
		// The log channel should have already been drained
		// There could stil be future messages sent into the channel, but the chance is really slim.
		timeout := time.After(100 * time.Millisecond)
		select {
		case log := <-logCh:
			t.Errorf("unexpected extra log: %+v", *log)
		case <-timeout:
		}
	}
}
