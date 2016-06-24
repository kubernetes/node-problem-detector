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
	"fmt"
	"strings"
	"time"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/types"
)

// Translator translates a log line into types.KernelLog, so that kernel monitor
// could parse it whatever the original format is.
type Translator interface {
	// Translate translates one log line into types.KernelLog.
	Translate(string) (*types.KernelLog, error)
}

// defaultTranslator works well for ubuntu and debian, but may not work well with
// other os distros. However it is easy to add a new translator for new os distro.
type defaultTranslator struct{}

// NewDefaultTranslator creates a default translator.
func NewDefaultTranslator() Translator {
	return &defaultTranslator{}
}

func (t *defaultTranslator) Translate(line string) (*types.KernelLog, error) {
	timestamp, message, err := t.parseLine(line)
	if err != nil {
		return nil, err
	}
	return &types.KernelLog{
		Timestamp: timestamp,
		Message:   message,
	}, nil
}

var (
	timestampLen  = 15
	messagePrefix = "]"
)

func (t *defaultTranslator) parseLine(line string) (time.Time, string, error) {
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
