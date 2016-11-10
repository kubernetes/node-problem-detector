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
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/logwatchers/types"
	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/util"
)

// Compiling go-systemd/sdjournald needs libsystemd-dev or libsystemd-journal-dev,
// which is not always available on all os distros and versions.
// So we add the build tag in this file, so that on unsupported os distro, user can
// disable this build tag.

// journaldWatcher is the log watcher for journald.
type journaldWatcher struct {
	journal *sdjournal.Journal
	cfg     types.WatcherConfig
	logCh   chan *kerntypes.KernelLog
	tomb    *util.Tomb
}

// NewJournaldWatcher is the create function of journald watcher.
func NewJournaldWatcher(cfg types.WatcherConfig) types.LogWatcher {
	return &journaldWatcher{
		cfg:  cfg,
		tomb: util.NewTomb(),
		// A capacity 1000 buffer should be enough
		logCh: make(chan *kerntypes.KernelLog, 1000),
	}
}

// Make sure NewJournaldWatcher is types.WatcherCreateFunc .
var _ types.WatcherCreateFunc = NewJournaldWatcher

// Watch starts the journal watcher.
func (j *journaldWatcher) Watch() (<-chan *kerntypes.KernelLog, error) {
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

// defaultJournalLogPath is the default path of journal log.
const defaultJournalLogPath = "/var/log/journal"

// getJournal returns a journal client.
func getJournal(cfg types.WatcherConfig) (*sdjournal.Journal, error) {
	// Get journal log path.
	path := defaultJournalLogPath
	if cfg.LogPath != "" {
		path = cfg.LogPath
	}
	// Get lookback duration.
	since, err := time.ParseDuration(cfg.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lookback duration %q: %v", cfg.Lookback, err)
	}
	// Get journal client from the log path.
	journal, err := sdjournal.NewJournalFromDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create journal client from path %q: %v", path, err)
	}
	// Seek journal client based on the lookback duration.
	start := time.Now().Add(-since)
	err = journal.SeekRealtimeUsec(uint64(start.UnixNano() / 1000))
	if err != nil {
		return nil, fmt.Errorf("failed to lookback %q: %v", since, err)
	}
	// TODO(random-liu): Make this configurable to support parsing other logs.
	kernelMatch := sdjournal.Match{
		Field: sdjournal.SD_JOURNAL_FIELD_TRANSPORT,
		Value: "kernel",
	}
	err = journal.AddMatch(kernelMatch.String())
	if err != nil {
		return nil, fmt.Errorf("failed to add log filter %#v: %v", kernelMatch, err)
	}
	return journal, nil
}

// translate translates journal entry into internal type.
func translate(entry *sdjournal.JournalEntry) *kerntypes.KernelLog {
	timestamp := time.Unix(0, int64(time.Duration(entry.RealtimeTimestamp)*time.Microsecond))
	message := strings.TrimSpace(entry.Fields["MESSAGE"])
	return &kerntypes.KernelLog{
		Timestamp: timestamp,
		Message:   message,
	}
}
