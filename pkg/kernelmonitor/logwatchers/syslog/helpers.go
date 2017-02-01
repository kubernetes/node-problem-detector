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
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"

	"github.com/google/cadvisor/utils/tail"
)

// translate translates the log line into internal type.
func translate(line string) (*kerntypes.KernelLog, error) {
	timestamp, message, err := parseLine(line)
	if err != nil {
		return nil, err
	}
	return &kerntypes.KernelLog{
		Timestamp: timestamp,
		Message:   message,
	}, nil
}

const (
	// timestampLen is the length of timestamp in syslog logging format.
	timestampLen = 15
	// messagePrefix is the character before real message.
	messagePrefix = "]"
)

// parseLine parses one log line into timestamp and message.
func parseLine(line string) (time.Time, string, error) {
	// Trim the spaces to make sure timestamp could be found
	line = strings.TrimSpace(line)
	if len(line) < timestampLen {
		return time.Time{}, "", fmt.Errorf("the line is too short: %q", line)
	}
	// Example line: Jan  1 00:00:00 hostname kernel: [0.000000] component: log message
	now := time.Now()
	// There is no time zone information in kernel log timestamp, apply the current time
	// zone.
	timestamp, err := time.ParseInLocation(time.Stamp, line[:timestampLen], time.Local)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("error parsing timestamp in line %q: %v", line, err)
	}
	// There is no year information in kernel log timestamp, apply the current year.
	// This could go wrong during looking back phase after kernel monitor is started,
	// and the old logs are generated in old year.
	timestamp = timestamp.AddDate(now.Year(), 0, 0)

	loc := strings.Index(line, messagePrefix)
	if loc == -1 {
		return timestamp, "", fmt.Errorf("can't find message prefix %q in line %q", messagePrefix, line)
	}
	message := strings.Trim(line[loc+1:], " ")

	return timestamp, message, nil
}

// defaultKernelLogPath the default path of syslog kernel log.
const defaultKernelLogPath = "/var/log/kern.log"

// getLogReader returns log reader for syslog log. Note that getLogReader doesn't look back
// to the rolled out logs.
func getLogReader(path string) (io.ReadCloser, error) {
	if path == "" {
		path = defaultKernelLogPath
	}
	// To handle log rotation, tail will not report error immediately if
	// the file doesn't exist. So we check file existence frist.
	// This could go wrong during mid-rotation. It should recover after
	// several restart when the log file is created again. The chance
	// is slim but we should still fix this in the future.
	// TODO(random-liu): Handle log missing during rotation.
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat the file %q: %v", path, err)
	}
	tail, err := tail.NewTail(path)
	if err != nil {
		return nil, fmt.Errorf("failed to tail the file %q: %v", path, err)
	}
	return tail, nil
}
