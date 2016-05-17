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
)

func TestGetStartPoint(t *testing.T) {
	testCases := []struct {
		log  string
		logs []types.KernelLog
		err  bool
	}{
		{
			// The start point is at the head of the log file.
			log: `kernel: [1.000000] 1
			kernel: [2.000000] 2
			kernel: [3.000000] 3
			`,
			logs: []types.KernelLog{
				{
					Timestamp: 1000000,
					Message:   "1",
				},
				{
					Timestamp: 2000000,
					Message:   "2",
				},
				{
					Timestamp: 3000000,
					Message:   "3",
				},
			},
		},
		{
			// The start point is in the middle of the log file.
			log: `kernel: [3.000000] 3
			kernel: [1.000000] 1
			kernel: [2.000000] 2
			`,
			logs: []types.KernelLog{
				{
					Timestamp: 1000000,
					Message:   "1",
				},
				{
					Timestamp: 2000000,
					Message:   "2",
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
			f.Close()
			os.Remove(f.Name())
		}()
		_, err = f.Write([]byte(test.log))
		if err != nil {
			t.Fatal(err)
		}
		w := NewKernelLogWatcher(WatcherConfig{KernelLogPath: f.Name()})
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
