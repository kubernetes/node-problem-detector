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
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/types"

	"github.com/pivotal-golang/clock/fakeclock"
)

func TestWatch(t *testing.T) {
	// now is a fake time
	now := time.Date(time.Now().Year(), time.January, 2, 3, 4, 5, 0, time.Local)
	fakeClock := fakeclock.NewFakeClock(now)
	testCases := []struct {
		log      string
		logs     []types.KernelLog
		lookback string
	}{
		{
			// The start point is at the head of the log file.
			log: `Jan  2 03:04:05 kernel: [0.000000] 1
			Jan  2 03:04:06 kernel: [1.000000] 2
			Jan  2 03:04:07 kernel: [2.000000] 3
			`,
			logs: []types.KernelLog{
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
			log: `Jan  2 03:04:04 kernel: [0.000000] 1
			Jan  2 03:04:05 kernel: [1.000000] 2
			Jan  2 03:04:06 kernel: [2.000000] 3
			`,
			logs: []types.KernelLog{
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
			log: `Jan  2 03:04:03 kernel: [0.000000] 1
			Jan  2 03:04:04 kernel: [1.000000] 2
			Jan  2 03:04:05 kernel: [2.000000] 3
			`,
			lookback: "1s",
			logs: []types.KernelLog{
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
		f, err := ioutil.TempFile("", "kernel_log_watcher_test")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			os.Remove(f.Name())
		}()
		_, err = f.Write([]byte(test.log))
		if err != nil {
			t.Fatal(err)
		}

		f.Close()
	
		w := NewKernelLogWatcher(WatcherConfig{KernelLogPath: f.Name(), Lookback: test.lookback})
		// Set the fake clock.
		w.(*kernelLogWatcher).clock = fakeClock
		logCh, err := w.Watch()
		if err != nil {
			t.Fatal(err)
		}
		defer w.Stop()
		for _, expected := range test.logs {
			got := <-logCh
			if !reflect.DeepEqual(&expected, got) {
				t.Errorf("case %d: expect %+v, got %+v", c+1, expected, *got)
			}
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
