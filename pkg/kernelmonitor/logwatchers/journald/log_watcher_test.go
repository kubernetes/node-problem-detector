// +build journald

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

package journald

import (
	"reflect"
	"testing"
	"time"

	"github.com/coreos/go-systemd/sdjournal"

	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
)

func TestTranslate(t *testing.T) {
	testCases := []struct {
		entry *sdjournal.JournalEntry
		log   *kerntypes.KernelLog
	}{
		{
			// has log message
			entry: &sdjournal.JournalEntry{
				Fields:            map[string]string{"MESSAGE": "log message"},
				RealtimeTimestamp: 123456789,
			},
			log: &kerntypes.KernelLog{
				Timestamp: time.Unix(0, 123456789*1000),
				Message:   "log message",
			},
		},
		{
			// no log message
			entry: &sdjournal.JournalEntry{
				Fields:            map[string]string{},
				RealtimeTimestamp: 987654321,
			},
			log: &kerntypes.KernelLog{
				Timestamp: time.Unix(0, 987654321*1000),
				Message:   "",
			},
		},
	}

	for c, test := range testCases {
		log := translate(test.entry)
		if !reflect.DeepEqual(test.log, log) {
			t.Errorf("case %d: expect %+v; got %+v", c+1, test.log, log)
		}
	}
}
