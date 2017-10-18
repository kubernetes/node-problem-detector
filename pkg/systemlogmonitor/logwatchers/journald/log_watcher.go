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
	"syscall"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

// Compiling go-systemd/sdjournald needs libsystemd-dev or libsystemd-journal-dev,
// which is not always available on all os distros and versions.
// So we add the build tag in this file, so that on unsupported os distro, user can
// disable this build tag.

// journaldWatcher is the log watcher for journald.
type journaldWatcher struct {
	journal *sdjournal.Journal
	cfg     types.WatcherConfig
	logCh   chan *logtypes.Log
	tomb    *tomb.Tomb
}

// NewJournaldWatcher is the create function of journald watcher.
func NewJournaldWatcher(cfg types.WatcherConfig) types.LogWatcher {
	return &journaldWatcher{
		cfg:  cfg,
		tomb: tomb.NewTomb(),
		// A capacity 1000 buffer should be enough
		logCh: make(chan *logtypes.Log, 1000),
	}
}

// Make sure NewJournaldWatcher is types.WatcherCreateFunc .
var _ types.WatcherCreateFunc = NewJournaldWatcher

// Watch starts the journal watcher.
func (j *journaldWatcher) Watch() (<-chan *logtypes.Log, error) {
	journal, err := getJournal(j.cfg)
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

		j.logCh <- translate(entry)
	}
}

const (
	// defaultJournalLogPath is the default path of journal log.
	defaultJournalLogPath = "/var/log/journal"

	// configSourceKey is the key of source configuration in the plugin configuration.
	configSourceKey = "source"
)

// getJournal returns a journal client.
func getJournal(cfg types.WatcherConfig) (*sdjournal.Journal, error) {
	// Get journal log path.
	path := defaultJournalLogPath
	if cfg.LogPath != "" {
		path = cfg.LogPath
	}
	// Get lookback duration.
	lookback, err := time.ParseDuration(cfg.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lookback duration %q: %v", cfg.Lookback, err)
	}
	// If the path doesn't present, NewJournalFromDir will create it instead of
	// returning error. So check the path existence ourselves.
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("failed to stat the log path %q: %v", path, err)
	}
	// Get journal client from the log path.
	journal, err := sdjournal.NewJournalFromDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create journal client from path %q: %v", path, err)
	}
	// Use system uptime if lookback duration is longer than it.
	// Ideally, we should use monotonic timestamp + boot id in journald. However, it doesn't seem
	// to work with go-system/journal package.
	// TODO(random-liu): Use monotonic timestamp + boot id.
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return nil, fmt.Errorf("failed to get system info: %v", err)
	}
	uptime := time.Duration(info.Uptime) * time.Second
	if lookback > uptime {
		lookback = uptime
		glog.Infof("Lookback changed to system uptime: %v", lookback)
	}
	// Seek journal client based on the lookback duration.
	start := time.Now().Add(-lookback)
	err = journal.SeekRealtimeUsec(uint64(start.UnixNano() / 1000))
	if err != nil {
		return nil, fmt.Errorf("failed to lookback %q: %v", lookback, err)
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
