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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

func TestTranslate(t *testing.T) {
	year := time.Now().Year()
	testCases := []struct {
		config map[string]string
		input  string
		err    bool
		log    *logtypes.Log
	}{
		{
			// missing year and timezone
			// "timestamp":       "^.{15}",
			// "message":         "kernel \\[.*\\] (.*)",
			// "timestampFormat": "Jan _2 15:04:05",
			config: getTestPluginConfig(),
			input:  "May  1 12:23:45 hostname kernel: [0.000000] component: log message",
			log: &logtypes.Log{
				Timestamp: time.Date(year, time.May, 1, 12, 23, 45, 0, time.Local),
				Message:   "component: log message",
			},
		},
		{
			// no log message
			config: getTestPluginConfig(),
			input:  "May 21 12:23:45 hostname kernel: [9.999999] ",
			log: &logtypes.Log{
				Timestamp: time.Date(year, time.May, 21, 12, 23, 45, 0, time.Local),
				Message:   "",
			},
		},
		{
			// the right square bracket is missing
			config: getTestPluginConfig(),
			input:  "May 21 12:23:45 hostname kernel: [9.999999 component: log message",
			err:    true,
		},
		{
			// contains full timestamp
			config: map[string]string{
				"timestamp":       "^time=\"(\\S*)\"",
				"message":         "msg=\"([^\n]*)\"",
				"timestampFormat": "2006-01-02T15:04:05.999999999-07:00",
			},
			input: `time="2017-02-01T17:58:34.999999999-08:00" level=error msg="test log line1\n test log line2"`,
			log: &logtypes.Log{
				Timestamp: time.Date(2017, 2, 1, 17, 58, 34, 999999999, time.FixedZone("PST", -8*3600)),
				Message:   `test log line1\n test log line2`,
			},
		},
	}

	for c, test := range testCases {
		t.Logf("TestCase #%d: %#v", c+1, test)
		trans := newTranslatorOrDie(test.config)
		log, err := trans.translate(test.input)
		if !test.err {
			require.NoError(t, err)
			// Use RFC3339Nano to make it easier for comparison.
			assert.Equal(t, test.log.Timestamp.Format(time.RFC3339Nano), log.Timestamp.Format(time.RFC3339Nano))
			assert.Equal(t, test.log.Message, log.Message)
		} else {
			require.Error(t, err)
		}
	}
}
