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
	"sync"
	"testing"
	"time"

	"github.com/euank/go-kmsg-parser/kmsgparser"
	"github.com/stretchr/testify/assert"
	testclock "k8s.io/utils/clock/testing"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

type mockKmsgParser struct {
	kmsgs          []kmsgparser.Message
	closeAfterSend bool
	closeCalled    bool
	mu             sync.Mutex
}

func (m *mockKmsgParser) SetLogger(kmsgparser.Logger) {}

func (m *mockKmsgParser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
	return nil
}

func (m *mockKmsgParser) WasCloseCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

func (m *mockKmsgParser) Parse() <-chan kmsgparser.Message {
	c := make(chan kmsgparser.Message)
	go func() {
		for _, msg := range m.kmsgs {
			c <- msg
		}
		if m.closeAfterSend {
			close(c)
		}
	}()
	return c
}

func (m *mockKmsgParser) SeekEnd() error { return nil }

func TestWatch(t *testing.T) {
	now := time.Date(time.Now().Year(), time.January, 2, 3, 4, 5, 0, time.Local)
	fakeClock := testclock.NewFakeClock(now)
	testCases := []struct {
		uptime   time.Duration
		lookback string
		delay    string
		log      *mockKmsgParser
		logs     []logtypes.Log
	}{
		{
			// The start point is at the head of the log file.
			uptime:   0,
			lookback: "0",
			delay:    "0",
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
		},
		{
			// The start point is in the middle of the log file.
			uptime:   0,
			lookback: "0",
			delay:    "0",
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
		},
		{
			// The start point is at the end of the log file, but we look back.
			uptime:   2 * time.Second,
			lookback: "1s",
			delay:    "0",
			log: &mockKmsgParser{kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(-2 * time.Second)},
				{Message: "2", Timestamp: now.Add(-1 * time.Second)},
				{Message: "3", Timestamp: now.Add(0 * time.Second)},
			}},
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
		{
			// The start point is at the end of the log file, but we look back up to start time.
			uptime:   time.Second,
			lookback: "3s",
			delay:    "0",
			log: &mockKmsgParser{kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(-3 * time.Second)},
				{Message: "2", Timestamp: now.Add(-2 * time.Second)},
				{Message: "3", Timestamp: now.Add(-1 * time.Second)},
				{Message: "4", Timestamp: now.Add(0 * time.Second)},
			}},
			logs: []logtypes.Log{
				{
					Timestamp: now.Add(-time.Second),
					Message:   "3",
				},
				{
					Timestamp: now,
					Message:   "4",
				},
			},
		},
	}
	for _, test := range testCases {
		w := NewKmsgWatcher(types.WatcherConfig{Lookback: test.lookback})
		w.(*kernelLogWatcher).startTime, _ = util.GetStartTime(fakeClock.Now(), test.uptime, test.lookback, test.delay)
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
		// There could still be future messages sent into the channel, but the chance is really slim.
		timeout := time.After(100 * time.Millisecond)
		select {
		case log := <-logCh:
			t.Errorf("unexpected extra log: %+v", *log)
		case <-timeout:
		}
	}
}

func TestWatcherStopsGracefullyOnTombStop(t *testing.T) {
	now := time.Now()

	mock := &mockKmsgParser{
		kmsgs: []kmsgparser.Message{
			{Message: "test message", Timestamp: now},
		},
		closeAfterSend: false, // Don't close, let tomb stop it
	}

	w := &kernelLogWatcher{
		cfg:        types.WatcherConfig{},
		startTime:  now.Add(-time.Second),
		tomb:       tomb.NewTomb(),
		logCh:      make(chan *logtypes.Log, 100),
		kmsgParser: mock,
	}

	logCh, err := w.Watch()
	assert.NoError(t, err)

	// Should receive the message
	select {
	case log := <-logCh:
		assert.Equal(t, "test message", log.Message)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for log message")
	}

	// Stop the watcher
	w.Stop()

	// Log channel should be closed after stop
	select {
	case _, ok := <-logCh:
		assert.False(t, ok, "log channel should be closed after Stop()")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for log channel to close after Stop()")
	}

	// Verify parser was closed
	assert.True(t, mock.WasCloseCalled(), "parser Close() should have been called")
}

func TestWatcherProcessesEmptyMessages(t *testing.T) {
	now := time.Now()

	mock := &mockKmsgParser{
		kmsgs: []kmsgparser.Message{
			{Message: "", Timestamp: now},
			{Message: "valid message", Timestamp: now.Add(time.Second)},
			{Message: "", Timestamp: now.Add(2 * time.Second)},
		},
		closeAfterSend: false,
	}

	w := &kernelLogWatcher{
		cfg:        types.WatcherConfig{},
		startTime:  now.Add(-time.Second),
		tomb:       tomb.NewTomb(),
		logCh:      make(chan *logtypes.Log, 100),
		kmsgParser: mock,
	}

	logCh, err := w.Watch()
	assert.NoError(t, err)

	// Should only receive the non-empty message
	select {
	case log := <-logCh:
		assert.Equal(t, "valid message", log.Message)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for log message")
	}

	// Stop the watcher and verify channel closes
	w.Stop()

	select {
	case _, ok := <-logCh:
		assert.False(t, ok, "log channel should be closed after Stop()")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for log channel to close")
	}
}

func TestWatcherTrimsMessageWhitespace(t *testing.T) {
	now := time.Now()

	mock := &mockKmsgParser{
		kmsgs: []kmsgparser.Message{
			{Message: "  message with spaces  ", Timestamp: now},
			{Message: "\ttabbed message\t", Timestamp: now.Add(time.Second)},
			{Message: "\n\nnewlines\n\n", Timestamp: now.Add(2 * time.Second)},
		},
		closeAfterSend: false,
	}

	w := &kernelLogWatcher{
		cfg:        types.WatcherConfig{},
		startTime:  now.Add(-time.Second),
		tomb:       tomb.NewTomb(),
		logCh:      make(chan *logtypes.Log, 100),
		kmsgParser: mock,
	}

	logCh, err := w.Watch()
	assert.NoError(t, err)

	expectedMessages := []string{"message with spaces", "tabbed message", "newlines"}

	for _, expected := range expectedMessages {
		select {
		case log := <-logCh:
			assert.Equal(t, expected, log.Message)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for message: %s", expected)
		}
	}

	// Stop the watcher and verify channel closes
	w.Stop()

	select {
	case _, ok := <-logCh:
		assert.False(t, ok, "log channel should be closed after Stop()")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for log channel to close")
	}
}
