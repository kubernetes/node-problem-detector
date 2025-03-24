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

package filelog

import (
	"os"
	"testing"
	"time"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"

	"github.com/stretchr/testify/assert"
	testclock "k8s.io/utils/clock/testing"
)

// getTestPluginConfig returns a plugin config for test. Use configuration for
// kernel log in test.
func getTestPluginConfig() map[string]string {
	return map[string]string{
		"timestamp":       "^.{15}",
		"message":         "kernel: \\[.*\\] (.*)",
		"timestampFormat": "Jan _2 15:04:05",
	}
}

func TestWatch(t *testing.T) {
	// now is a fake time
	now := time.Date(time.Now().Year(), time.January, 2, 3, 4, 5, 0, time.Local)
	fakeClock := testclock.NewFakeClock(now)
	testCases := []struct {
		uptime   time.Duration
		lookback string
		delay    string
		log      string
		logs     []logtypes.Log
	}{
		{
			// The start point is at the head of the log file.
			uptime:   0,
			lookback: "0",
			delay:    "0",
			log: `Jan  2 03:04:05 kernel: [0.000000] 1
Jan  2 03:04:06 kernel: [1.000000] 2
Jan  2 03:04:07 kernel: [2.000000] 3
			`,
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
			log: `Jan  2 03:04:04 kernel: [0.000000] 1
Jan  2 03:04:05 kernel: [1.000000] 2
Jan  2 03:04:06 kernel: [2.000000] 3
			`,
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
			log: `Jan  2 03:04:03 kernel: [0.000000] 1
Jan  2 03:04:04 kernel: [1.000000] 2
Jan  2 03:04:05 kernel: [2.000000] 3
			`,
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
			// The start point is at the end of the log file, we look back, but
			// system rebooted at in the middle of the log file.
			uptime:   time.Second,
			lookback: "2s",
			delay:    "0",
			log: `Jan  2 03:04:03 kernel: [0.000000] 1
Jan  2 03:04:04 kernel: [1.000000] 2
Jan  2 03:04:05 kernel: [2.000000] 3
			`,
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
	for c, test := range testCases {
		t.Logf("TestCase #%d: %#v", c+1, test)
		f, err := os.CreateTemp("", "log_watcher_test")
		assert.NoError(t, err)
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()
		_, err = f.Write([]byte(test.log))
		assert.NoError(t, err)

		w := NewSyslogWatcherOrDie(types.WatcherConfig{
			Plugin:       "filelog",
			PluginConfig: getTestPluginConfig(),
			LogPath:      f.Name(),
			Lookback:     test.lookback,
		})
		// Set the startTime.
		w.(*filelogWatcher).startTime, _ = util.GetStartTime(fakeClock.Now(), test.uptime, test.lookback, test.delay)
		logCh, err := w.Watch()
		assert.NoError(t, err)
		defer w.Stop()
		for _, expected := range test.logs {
			select {
			case got := <-logCh:
				assert.Equal(t, &expected, got)
			case <-time.After(30 * time.Second):
				t.Errorf("timeout waiting for log")
			}
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

func TestFilterSkipList(t *testing.T) {
	s := &filelogWatcher{
		cfg: types.WatcherConfig{
			SkipList: []string{
				" audit:", " kubelet:",
			},
		},
	}
	testcase := []struct {
		log    string
		expect bool
	}{
		{
			log:    `Jan  2 03:04:03 kernel: [0.000000] 1`,
			expect: false,
		},
		{
			log:    `Jan  2 03:04:04 audit: [1.000000] 2`,
			expect: true,
		},
		{
			log:    `Jan  2 03:04:05 kubelet: [2.000000] 3`,
			expect: true,
		},
	}
	for i, test := range testcase {
		if s.filterSkipList(test.log) != test.expect {
			t.Errorf("test case %d: expect %v but got %v", i, test.expect, s.filterSkipList(test.log))
		}
	}
}
