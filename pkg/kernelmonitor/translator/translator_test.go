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

package translator

import (
	"testing"
	"time"
)

func TestDefaultTranslator(t *testing.T) {
	tr := NewDefaultTranslator()
	year := time.Now().Year()
	testCases := []struct {
		input     string
		err       bool
		timestamp time.Time
		message   string
	}{
		{
			input:     "May  1 12:23:45 hostname kernel: [0.000000] component: log message",
			timestamp: time.Date(year, time.May, 1, 12, 23, 45, 0, time.Local),
			message:   "component: log message",
		},
		{
			// no log message
			input:     "May 21 12:23:45 hostname kernel: [9.999999]",
			timestamp: time.Date(year, time.May, 21, 12, 23, 45, 0, time.Local),
			message:   "",
		},
		{
			// the right square bracket is missing
			input: "May 21 12:23:45 hostname kernel: [9.999999 component: log message",
			err:   true,
		},
	}

	for c, test := range testCases {
		log, err := tr.Translate(test.input)
		if test.err {
			if err == nil {
				t.Errorf("case %d: expect error should occur, got %+v, %v", c+1, log, err)
			}
			continue
		}
		if test.timestamp != log.Timestamp || test.message != log.Message {
			t.Errorf("case %d: expect %v, %q; got %v, %q", c+1, test.timestamp, test.message, log.Timestamp, log.Message)
		}
	}
}
