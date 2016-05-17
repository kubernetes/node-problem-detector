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
)

func TestDefaultTranslator(t *testing.T) {
	tr := NewDefaultTranslator()

	testCases := []struct {
		input     string
		err       bool
		timestamp int64
		message   string
	}{
		{
			input:     "Jan  1 00:00:00 hostname kernel: [9.999999] component: log message",
			timestamp: 9999999,
			message:   "component: log message",
		},
		{
			input:     "Jan  1 00:00:00 hostname kernel: [9.999999]",
			timestamp: 9999999,
			message:   "",
		},
		{
			input: "Jan  1 00:00:00 hostname kernel: [9.999999 component: log message",
			err:   true,
		},
		{
			input: "Jan  1 00:00:00 hostname user: [9.999999] component: log message",
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
			t.Errorf("case %d: expect timestamp: %d, message: %q; got %+v", c+1, test.timestamp, test.message, log)
		}
	}
}
