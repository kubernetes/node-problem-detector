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
	"io"
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/plugins/types"
	kmtypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/util"
)

// Compiling go-systemd/sdjournald needs libsystemd-dev or libsystemd-journal-dev,
// which is not always available on all os distros and versions.
// So we add the build tag in this file, so that on unsupported os distro, user can
// disable this build tag.

// journaldPlugin is the log parsing plugin for journald.
type journaldPlugin struct {
	io.ReadCloser
}

// NewJournaldPlugin is the create function of journald plugin.
func NewJournaldPlugin(cfg types.Config) (types.Plugin, error) {
	r, err := getJournalLogReader(cfg)
	if err != nil {
		return nil, err
	}
	return &journaldPlugin{r}, nil
}

// Make sure NewJournaldPlugin is types.PluginCreateFunc.
var _ types.PluginCreateFunc = NewJournaldPlugin

// Translate translates the log line into internal type.
func (s *journaldPlugin) Translate(line string) (*kmtypes.KernelLog, error) {
	timestamp, message, err := parseLine(line)
	if err != nil {
		return nil, err
	}
	return &kmtypes.KernelLog{
		Timestamp: timestamp,
		Message:   message,
	}, nil
}

const (
	// messagePrefix is the prefix before real message. Before the prefix
	// should be the timestamp.
	messagePrefix = "MESSAGE="
	// default golang time.String() format.
	format = "2006-01-02 15:04:05.999999999 -0700 MST"
)

// parseLine parses one log line into timestamp and message.
func parseLine(line string) (time.Time, string, error) {
	loc := strings.Index(line, messagePrefix)
	if loc == -1 {
		return time.Time{}, "", fmt.Errorf("can't find message prefix %q in line %q", messagePrefix, line)
	}
	// Example line: 2016-11-18 22:55:08.279282 +0000 UTC MESSAGE=log message
	timestamp, err := time.Parse(format, strings.TrimSpace(line[:loc]))
	if err != nil {
		return time.Time{}, "", fmt.Errorf("error parsing timestamp in line %q: %v", line, err)
	}
	message := strings.TrimSpace(line[loc+len(messagePrefix):])
	return timestamp, message, nil
}

const (
	// defaultJournalLogPath is the default path of journal log.
	defaultJournalLogPath = "/var/log/journal"
)

// getJournalLogReader is the log reader getter for journal based kernel log.
func getJournalLogReader(cfg types.Config) (io.ReadCloser, error) {
	path := defaultJournalLogPath
	if cfg.LogPath != "" {
		path = cfg.LogPath
	}
	since, err := util.ParseDuration(cfg.Lookback)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lookback duration %q: %v", cfg.Lookback, err)
	}
	r, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		Path:  path,
		Since: -since,
		Matches: []sdjournal.Match{
			{
				Field: sdjournal.SD_JOURNAL_FIELD_TRANSPORT,
				Value: "kernel",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error opening journal: %v", err)
	}
	if r == nil {
		return nil, fmt.Errorf("got a nil journal log reader")
	}
	return r, nil
}
