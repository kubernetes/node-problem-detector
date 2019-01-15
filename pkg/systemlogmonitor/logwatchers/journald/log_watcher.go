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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

// Compiling go-systemd/sdjournald needs libsystemd-dev or libsystemd-journal-dev,
// which is not always available on all os distros and versions.
// So we add the build tag in this file, so that on unsupported os distro, user can
// disable this build tag.

// journaldWatcher is the log watcher for journald.
type journaldWatcher struct {
	journal   *sdjournal.Journal
	cfg       types.WatcherConfig
	startTime time.Time
	logCh     chan *logtypes.Log
	tomb      *tomb.Tomb
}

// NewJournaldWatcher is the create function of journald watcher.
func NewJournaldWatcher(cfg types.WatcherConfig) types.LogWatcher {
	uptime, err := util.GetUptimeDuration()
	if err != nil {
		glog.Fatalf("failed to get uptime: %v", err)
	}
	startTime, err := util.GetStartTime(time.Now(), uptime, cfg.Lookback, cfg.Delay)
	if err != nil {
		glog.Fatalf("failed to get start time: %v", err)
	}

	return &journaldWatcher{
		cfg:       cfg,
		startTime: startTime,
		tomb:      tomb.NewTomb(),
		// A capacity 1000 buffer should be enough
		logCh: make(chan *logtypes.Log, 1000),
	}
}

// Make sure NewJournaldWatcher is types.WatcherCreateFunc .
var _ types.WatcherCreateFunc = NewJournaldWatcher

// Watch starts the journal watcher.
func (j *journaldWatcher) Watch() (<-chan *logtypes.Log, error) {
	journal, err := getJournal(j.cfg, j.startTime)
	if err != nil {
		return nil, err
	}
	j.journal = journal
	glog.Info("Start watching journald")
	go j.watchLoop()
	return j.logCh, nil
}

// Stop stops the journald watcher.
func (j *journaldWatcher) Stop() {
	j.tomb.Stop()
}

// waitLogTimeout is the timeout waiting for new log.
const waitLogTimeout = 5 * time.Second

// watchLoop is the main watch loop of journald watcher.
func (j *journaldWatcher) watchLoop() {
	startTimestamp := timeToJournalTimestamp(j.startTime)
	defer func() {
		if err := j.journal.Close(); err != nil {
			glog.Errorf("Failed to close journal client: %v", err)
		}
		j.tomb.Done()
	}()
	for {
		select {
		case <-j.tomb.Stopping():
			glog.Infof("Stop watching journald")
			return
		default:
		}
		// Get next log entry.
		n, err := j.journal.Next()
		if err != nil {
			glog.Errorf("Failed to get next journal entry: %v", err)
			continue
		}
		// If next reaches the end, wait for waitLogTimeout.
		if n == 0 {
			j.journal.Wait(waitLogTimeout)
			continue
		}

		entry, err := j.journal.GetEntry()
		if err != nil {
			glog.Errorf("failed to get journal entry: %v", err)
			continue
		}

		if entry.RealtimeTimestamp < startTimestamp {
			glog.V(5).Infof("Throwing away journal entry %q before start time: %v < %v",
				entry.Fields[sdjournal.SD_JOURNAL_FIELD_MESSAGE], entry.RealtimeTimestamp, startTimestamp)
			continue
		}

		j.logCh <- translate(entry)
	}
}

const (
	// configSourceKey is the key of source configuration in the plugin configuration.
	configSourceKey = "source"
)

// getJournal returns a journal client.
func getJournal(cfg types.WatcherConfig, startTime time.Time) (*sdjournal.Journal, error) {
	var journal *sdjournal.Journal
	var err error
	if cfg.LogPath == "" {
		journal, err = sdjournal.NewJournal()
		if err != nil {
			return nil, fmt.Errorf("failed to create journal client from default log path: %v", err)
		}
		glog.Info("unspecified log path so using systemd default")
	} else {
		// If the path doesn't exist, NewJournalFromDir will
		// create it instead of returning error. So check the
		// path existence ourselves.
		if _, err = os.Stat(cfg.LogPath); err != nil {
			return nil, fmt.Errorf("failed to stat the log path %q: %v", cfg.LogPath, err)
		}
		// Get journal client from the log path.
		journal, err = sdjournal.NewJournalFromDir(cfg.LogPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create journal client from path %q: %v", cfg.LogPath, err)
		}
	}
	// Seek journal client based on startTime.
	seekTime := startTime
	now := time.Now()
	if now.Before(seekTime) {
		seekTime = now
	}
	err = journal.SeekRealtimeUsec(timeToJournalTimestamp(seekTime))
	if err != nil {
		return nil, fmt.Errorf("failed to seek journal at %v (now %v): %v", seekTime, now, err)
	}
	// Empty source is not allowed and treated as an error.
	source := cfg.PluginConfig[configSourceKey]
	if source == "" {
		return nil, fmt.Errorf("failed to filter journal log, empty source is not allowed")
	}
	match := sdjournal.Match{
		Field: sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER,
		Value: source,
	}
	err = journal.AddMatch(match.String())
	if err != nil {
		return nil, fmt.Errorf("failed to add log filter %#v: %v", match, err)
	}
	return journal, nil
}

// translate translates journal entry into internal type.
func translate(entry *sdjournal.JournalEntry) *logtypes.Log {
	timestamp := time.Unix(0, int64(time.Duration(entry.RealtimeTimestamp)*time.Microsecond))
	message := strings.TrimSpace(entry.Fields["MESSAGE"])
	return &logtypes.Log{
		Timestamp: timestamp,
		Message:   message,
	}
}

func timeToJournalTimestamp(t time.Time) uint64 {
	return uint64(t.UnixNano() / 1000)
}
