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
	"strconv"
	"strings"

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
	timestr, message, err := parseLine(line)
	if err != nil {
		return nil, err
	}
	timestamp, err := parseTimestamp(timestr)
	if err != nil {
		return nil, err
	}
	return &types.KernelLog{
		Timestamp: timestamp,
		Message:   message,
	}, nil
}

func parseLine(line string) (string, string, error) {
	// Example line: Jan  1 00:00:00 hostname kernel: [0.000000] component: log message
	timestampPrefix := "kernel: ["
	timestampSuffix := "]"
	idx := strings.Index(line, timestampPrefix)
	if idx == -1 {
		return "", "", fmt.Errorf("can't find timestamp prefix %q in line %q", timestampPrefix, line)
	}
	line = line[idx+len(timestampPrefix):]

	idx = strings.Index(line, timestampSuffix)
	if idx == -1 {
		return "", "", fmt.Errorf("can't find timestamp suffix %q in line %q", timestampSuffix, line)
	}

	timestamp := strings.Trim(line[:idx], " ")
	message := strings.Trim(line[idx+1:], " ")

	return timestamp, message, nil
}

func parseTimestamp(timestamp string) (int64, error) {
	f, err := strconv.ParseFloat(timestamp, 64)
	if err != nil {
		return 0, err
	}
	// seconds to microseconds
	return int64(f * 1000000), nil
}
