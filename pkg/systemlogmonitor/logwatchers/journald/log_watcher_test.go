//go:build journald
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
	"runtime"
	"testing"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

func TestTranslate(t *testing.T) {
	testCases := []struct {
		entry *sdjournal.JournalEntry
		log   *logtypes.Log
	}{
		{
			// has log message
			entry: &sdjournal.JournalEntry{
				Fields:            map[string]string{"MESSAGE": "log message"},
				RealtimeTimestamp: 123456789,
			},
			log: &logtypes.Log{
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
			log: &logtypes.Log{
				Timestamp: time.Unix(0, 987654321*1000),
				Message:   "",
			},
		},
	}

	for c, test := range testCases {
		t.Logf("TestCase #%d: %#v", c+1, test)
		assert.Equal(t, test.log, translate(test.entry))
	}
}

func TestGoroutineLeak(t *testing.T) {
	original := runtime.NumGoroutine()
	w := NewJournaldWatcher(types.WatcherConfig{
		Plugin:       "journald",
		PluginConfig: map[string]string{"source": "not-exist-service"},
		LogPath:      "/not/exist/path",
		Lookback:     "10m",
	})
	_, err := w.Watch()
	assert.Error(t, err)
	assert.Equal(t, original, runtime.NumGoroutine())
}
