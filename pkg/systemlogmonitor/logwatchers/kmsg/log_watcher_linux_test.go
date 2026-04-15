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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/euank/go-kmsg-parser/kmsgparser"
	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

type mockKmsgParser struct {
	kmsgs          []kmsgparser.Message
	closeAfterSend bool
	// closeClosesChannel, when true, causes Close() to close the Parse()
	// output channel, mirroring the real kmsgparser's behavior where
	// closing the underlying reader terminates its read goroutine and the
	// deferred close(output) fires.
	closeClosesChannel bool
	// seekEndErr, if non-nil, is returned from SeekEnd().
	seekEndErr error

	mu           sync.Mutex
	closeCount   int
	seekEndCount int
	parseCount   int
	done         chan struct{}
	doneOnce     sync.Once
}

func (m *mockKmsgParser) SetLogger(kmsgparser.Logger) {}

func (m *mockKmsgParser) Close() error {
	m.mu.Lock()
	m.closeCount++
	if m.closeClosesChannel && m.done == nil {
		m.done = make(chan struct{})
	}
	done := m.done
	m.mu.Unlock()

	if m.closeClosesChannel {
		m.doneOnce.Do(func() { close(done) })
	}
	return nil
}

func (m *mockKmsgParser) CloseCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCount
}

func (m *mockKmsgParser) Parse() <-chan kmsgparser.Message {
	m.mu.Lock()
	m.parseCount++
	c := make(chan kmsgparser.Message)
	if m.done == nil {
		m.done = make(chan struct{})
	}
	done := m.done
	m.mu.Unlock()

	go func() {
		for _, msg := range m.kmsgs {
			select {
			case c <- msg:
			case <-done:
				// Close() was called mid-send. Mirror real kmsgparser:
				// close the output so the consumer sees ok == false.
				close(c)
				return
			}
		}
		if m.closeAfterSend {
			close(c)
			return
		}
		if m.closeClosesChannel {
			// Wait for Close() to signal, then close the output channel.
			<-done
			close(c)
		}
	}()
	return c
}

func (m *mockKmsgParser) SeekEnd() error {
	m.mu.Lock()
	m.seekEndCount++
	m.mu.Unlock()
	return m.seekEndErr
}

func (m *mockKmsgParser) SeekEndCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.seekEndCount
}

func (m *mockKmsgParser) ParseCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.parseCount
}

func TestWatch(t *testing.T) {
	now := time.Date(time.Now().Year(), time.January, 2, 3, 4, 5, 0, time.Local)
	testCases := []struct {
		name     string
		uptime   time.Duration
		lookback string
		delay    string
		kmsgs    []kmsgparser.Message
		logs     []logtypes.Log
	}{
		{
			name:     "start at head of log",
			uptime:   0,
			lookback: "0",
			delay:    "0",
			kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(0 * time.Second)},
				{Message: "2", Timestamp: now.Add(1 * time.Second)},
				{Message: "3", Timestamp: now.Add(2 * time.Second)},
			},
			logs: []logtypes.Log{
				{Timestamp: now, Message: "1"},
				{Timestamp: now.Add(time.Second), Message: "2"},
				{Timestamp: now.Add(2 * time.Second), Message: "3"},
			},
		},
		{
			name:     "start in middle of log",
			uptime:   0,
			lookback: "0",
			delay:    "0",
			kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(-1 * time.Second)},
				{Message: "2", Timestamp: now.Add(0 * time.Second)},
				{Message: "3", Timestamp: now.Add(1 * time.Second)},
			},
			logs: []logtypes.Log{
				{Timestamp: now, Message: "2"},
				{Timestamp: now.Add(time.Second), Message: "3"},
			},
		},
		{
			name:     "start at end with lookback",
			uptime:   2 * time.Second,
			lookback: "1s",
			delay:    "0",
			kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(-2 * time.Second)},
				{Message: "2", Timestamp: now.Add(-1 * time.Second)},
				{Message: "3", Timestamp: now.Add(0 * time.Second)},
			},
			logs: []logtypes.Log{
				{Timestamp: now.Add(-time.Second), Message: "2"},
				{Timestamp: now, Message: "3"},
			},
		},
		{
			name:     "lookback bounded by uptime",
			uptime:   time.Second,
			lookback: "3s",
			delay:    "0",
			kmsgs: []kmsgparser.Message{
				{Message: "1", Timestamp: now.Add(-3 * time.Second)},
				{Message: "2", Timestamp: now.Add(-2 * time.Second)},
				{Message: "3", Timestamp: now.Add(-1 * time.Second)},
				{Message: "4", Timestamp: now.Add(0 * time.Second)},
			},
			logs: []logtypes.Log{
				{Timestamp: now.Add(-time.Second), Message: "3"},
				{Timestamp: now, Message: "4"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startTime, err := util.GetStartTime(now, tc.uptime, tc.lookback, tc.delay)
			if err != nil {
				t.Fatal(err)
			}
			w := &kernelLogWatcher{
				cfg:        types.WatcherConfig{Lookback: tc.lookback},
				startTime:  startTime,
				tomb:       tomb.NewTomb(),
				logCh:      make(chan *logtypes.Log, 100),
				kmsgParser: &mockKmsgParser{kmsgs: tc.kmsgs},
			}
			logCh, err := w.Watch()
			if err != nil {
				t.Fatal(err)
			}
			defer w.Stop()
			for _, expected := range tc.logs {
				got := <-logCh
				assert.Equal(t, &expected, got)
			}
			// The log channel should have already been drained.
			select {
			case log := <-logCh:
				t.Errorf("unexpected extra log: %+v", *log)
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}

// TestWatchReturnsErrorWhenNewParserFails verifies that Watch() propagates
// the newParser factory's error when no parser has been pre-set on the
// watcher, instead of starting watchLoop with a nil parser.
func TestWatchReturnsErrorWhenNewParserFails(t *testing.T) {
	w := &kernelLogWatcher{
		cfg:   types.WatcherConfig{},
		tomb:  tomb.NewTomb(),
		logCh: make(chan *logtypes.Log, 100),
		// kmsgParser is deliberately left nil so Watch() must call newParser.
		newParser: func() (kmsgparser.Parser, error) {
			return nil, fmt.Errorf("simulated newParser failure")
		},
	}

	logCh, err := w.Watch()
	assert.Error(t, err)
	assert.Nil(t, logCh)
	assert.Contains(t, err.Error(), "failed to create kmsg parser")
	assert.Contains(t, err.Error(), "simulated newParser failure")
}

// TestStopClosesParserCleanly verifies that Stop() shuts the watcher down
// cleanly: the log channel is closed, the parser's Close() is called
// exactly once (regression guard for the fix that removed Close() from
// Stop()), and the watch loop's restart path is not triggered (the
// injected newParser must not be called).
//
// The "realistic parser" case is the one that exercises the exact
// production bug: when Close() closes the Parse() output channel, the
// buggy version of Stop() drove watchLoop down the restart path during
// intentional shutdown.
func TestStopClosesParserCleanly(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name               string
		kmsgs              []kmsgparser.Message
		closeClosesChannel bool
	}{
		{
			name: "single message",
			kmsgs: []kmsgparser.Message{
				{Message: "test message", Timestamp: now},
			},
		},
		{
			name: "multiple messages",
			kmsgs: []kmsgparser.Message{
				{Message: "msg-1", Timestamp: now},
				{Message: "msg-2", Timestamp: now.Add(time.Second)},
			},
		},
		{
			name: "realistic parser (Close closes Parse channel)",
			kmsgs: []kmsgparser.Message{
				{Message: "test message", Timestamp: now},
			},
			closeClosesChannel: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockKmsgParser{
				kmsgs:              tc.kmsgs,
				closeClosesChannel: tc.closeClosesChannel,
			}

			factoryCalls := 0
			w := &kernelLogWatcher{
				cfg:        types.WatcherConfig{},
				startTime:  now.Add(-time.Minute),
				tomb:       tomb.NewTomb(),
				logCh:      make(chan *logtypes.Log, 100),
				kmsgParser: mock,
				newParser: func() (kmsgparser.Parser, error) {
					factoryCalls++
					return nil, fmt.Errorf("factory should not be called during clean Stop")
				},
			}

			logCh, err := w.Watch()
			assert.NoError(t, err)

			// Drain all messages so the parser goroutine has finished sending
			// before we Stop. Keeps the test deterministic and focused on Stop.
			for _, expected := range tc.kmsgs {
				select {
				case got := <-logCh:
					assert.Equal(t, expected.Message, got.Message)
				case <-time.After(time.Second):
					t.Fatalf("timeout waiting for message %q", expected.Message)
				}
			}

			w.Stop()

			select {
			case _, ok := <-logCh:
				assert.False(t, ok, "log channel should be closed after Stop()")
			case <-time.After(time.Second):
				t.Fatal("timeout waiting for log channel to close after Stop()")
			}

			assert.Equal(t, 0, factoryCalls, "newParser should not be called during clean Stop")
			assert.Equal(t, 1, mock.CloseCallCount(), "parser Close() should be called exactly once after Stop()")
		})
	}
}

// TestWatcherRestartsOnUnexpectedChannelClose exercises the legitimate
// restart path from #1192: when the Parse() channel closes outside of a
// Stop() (e.g. the underlying /dev/kmsg reader hit an error), watchLoop
// must recover by calling newParser() and continue delivering messages
// from the fresh parser. Prior to the factory injection there was no
// test coverage for this path.
func TestWatcherRestartsOnUnexpectedChannelClose(t *testing.T) {
	now := time.Now()

	// First parser: sends one message then closes its Parse channel to
	// simulate an underlying reader error.
	first := &mockKmsgParser{
		kmsgs: []kmsgparser.Message{
			{Message: "before-restart", Timestamp: now},
		},
		closeAfterSend: true,
	}

	// Second parser: delivers messages after the restart.
	second := &mockKmsgParser{
		kmsgs: []kmsgparser.Message{
			{Message: "after-restart", Timestamp: now.Add(time.Second)},
		},
	}

	factoryCalls := 0
	w := &kernelLogWatcher{
		cfg:        types.WatcherConfig{},
		startTime:  now.Add(-time.Minute),
		tomb:       tomb.NewTomb(),
		logCh:      make(chan *logtypes.Log, 100),
		kmsgParser: first,
		newParser: func() (kmsgparser.Parser, error) {
			factoryCalls++
			return second, nil
		},
	}

	logCh, err := w.Watch()
	assert.NoError(t, err)

	select {
	case log := <-logCh:
		assert.Equal(t, "before-restart", log.Message)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message before restart")
	}

	select {
	case log := <-logCh:
		assert.Equal(t, "after-restart", log.Message)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message after restart")
	}

	w.Stop()

	select {
	case _, ok := <-logCh:
		assert.False(t, ok, "log channel should be closed after Stop()")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for log channel to close after Stop()")
	}

	assert.Equal(t, 1, factoryCalls, "newParser should have been called exactly once for the restart")
	assert.Equal(t, 1, first.CloseCallCount(), "first parser should be closed once by the restart path")
	assert.Equal(t, 1, second.CloseCallCount(), "second parser should be closed once by Stop's defer")
	assert.Equal(t, 1, first.ParseCallCount(), "Parse should be called once on the initial parser")
	assert.Equal(t, 1, second.ParseCallCount(), "Parse should be called once on the restart parser after a successful SeekEnd")
	// The initial parser uses the watcher's startTime/lookback semantics
	// and must NOT be seeked. The restart parser, by contrast, must be
	// seeked to the end of the ring buffer to avoid replaying messages
	// that were already processed before the restart.
	assert.Equal(t, 0, first.SeekEndCallCount(), "SeekEnd should not be called on the initial parser")
	assert.Equal(t, 1, second.SeekEndCallCount(), "SeekEnd should be called once on the restart parser")
}

// TestWatcherProcessesMessageContent verifies watchLoop's per-message
// handling: empty messages are dropped, and surrounding whitespace is
// trimmed before forwarding.
func TestWatcherProcessesMessageContent(t *testing.T) {
	now := time.Now()

	cases := []struct {
		name     string
		kmsgs    []kmsgparser.Message
		expected []string
	}{
		{
			name: "drops empty messages",
			kmsgs: []kmsgparser.Message{
				{Message: "", Timestamp: now},
				{Message: "valid message", Timestamp: now.Add(time.Second)},
				{Message: "", Timestamp: now.Add(2 * time.Second)},
			},
			expected: []string{"valid message"},
		},
		{
			name: "trims surrounding whitespace",
			kmsgs: []kmsgparser.Message{
				{Message: "  message with spaces  ", Timestamp: now},
				{Message: "\ttabbed message\t", Timestamp: now.Add(time.Second)},
				{Message: "\n\nnewlines\n\n", Timestamp: now.Add(2 * time.Second)},
			},
			expected: []string{"message with spaces", "tabbed message", "newlines"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &kernelLogWatcher{
				cfg:        types.WatcherConfig{},
				startTime:  now.Add(-time.Minute),
				tomb:       tomb.NewTomb(),
				logCh:      make(chan *logtypes.Log, 100),
				kmsgParser: &mockKmsgParser{kmsgs: tc.kmsgs},
			}

			logCh, err := w.Watch()
			assert.NoError(t, err)
			defer w.Stop()

			for _, want := range tc.expected {
				select {
				case got := <-logCh:
					assert.Equal(t, want, got.Message)
				case <-time.After(time.Second):
					t.Fatalf("timeout waiting for message %q", want)
				}
			}
		})
	}
}
