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

	kmtypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
)

func TestTranslator(t *testing.T) {
	p := journaldPlugin{}
	testCases := []struct {
		input string
		err   bool
		log   *kmtypes.KernelLog
	}{
		{
			input: "2016-11-18 22:55:08 +0000 UTC MESSAGE=log message",
			log: &kmtypes.KernelLog{
				Timestamp: time.Date(2016, time.November, 18, 22, 55, 8, 0, time.UTC),
				Message:   "log message",
			},
		},
		{
			// no log message
			input: "2015-10-22 22:22:22.666666666 +0000 UTC MESSAGE=",
			log: &kmtypes.KernelLog{
				Timestamp: time.Date(2015, time.October, 22, 22, 22, 22, 666666666, time.UTC),
				Message:   "",
			},
		},
		{
			// the message prefix is missing
			input: "2014-11-11 11:11:11 +0000 UTC MESSAGE log message",
			err:   true,
		},
	}

	for c, test := range testCases {
		log, err := p.Translate(test.input)
		if (err != nil) != test.err {
			t.Errorf("case %d: error assertion failed, got log: %+v, error: %v", c+1, log, err)
			continue
		}
		if !reflect.DeepEqual(test.log, log) {
			t.Errorf("case %d: expect %+v; got %+v", c+1, test.log, log)
		}
	}
}
