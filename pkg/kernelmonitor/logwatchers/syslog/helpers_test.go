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

package syslog

import (
	"reflect"
	"testing"
	"time"

	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
)

func TestTranslate(t *testing.T) {
	year := time.Now().Year()
	testCases := []struct {
		input string
		err   bool
		log   *kerntypes.KernelLog
	}{
		{
			input: "May  1 12:23:45 hostname kernel: [0.000000] component: log message",
			log: &kerntypes.KernelLog{
				Timestamp: time.Date(year, time.May, 1, 12, 23, 45, 0, time.Local),
				Message:   "component: log message",
			},
		},
		{
			// no log message
			input: "May 21 12:23:45 hostname kernel: [9.999999]",
			log: &kerntypes.KernelLog{
				Timestamp: time.Date(year, time.May, 21, 12, 23, 45, 0, time.Local),
				Message:   "",
			},
		},
		{
			// the right square bracket is missing
			input: "May 21 12:23:45 hostname kernel: [9.999999 component: log message",
			err:   true,
		},
	}

	for c, test := range testCases {
		log, err := translate(test.input)
		if (err != nil) != test.err {
			t.Errorf("case %d: error assertion failed, got log: %+v, error: %v", c+1, log, err)
			continue
		}
		if !reflect.DeepEqual(test.log, log) {
			t.Errorf("case %d: expect %+v; got %+v", c+1, test.log, log)
		}
	}
}
