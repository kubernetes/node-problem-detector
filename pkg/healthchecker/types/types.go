/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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

package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultCoolDownTime       = 2 * time.Minute
	DefaultHealthCheckTimeout = 10 * time.Second
	CmdTimeout                = 10 * time.Second
	UptimeTimeLayout          = "Mon 2006-01-02 15:04:05 UTC"
	LogParsingTimeLayout      = "2006-01-02 15:04:05"

	DefaultCriCtl        = "/usr/bin/crictl"
	DefaultCriSocketPath = "unix:///var/run/containerd/containerd.sock"

	KubeletComponent  = "kubelet"
	CRIComponent      = "cri"
	DockerComponent   = "docker"
	ContainerdService = "containerd"

	KubeletHealthCheckEndpoint = "http://127.0.0.1:10248/healthz"

	LogPatternFlagSeparator = ":"
)

type HealthChecker interface {
	CheckHealth() (bool, error)
}

// LogPatternFlag defines the flag for log pattern health check.
// It contains a map of <log pattern> to <failure threshold for the pattern>
type LogPatternFlag struct {
	logPatternCountMap map[string]int
}

// String implements the String function for flag.Value interface
func (lpf *LogPatternFlag) String() string {
	result := ""
	for k, v := range lpf.logPatternCountMap {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("%v:%v", k, v)
	}
	return result
}

// Set implements the Set function for flag.Value interface
func (lpf *LogPatternFlag) Set(value string) error {
	if lpf.logPatternCountMap == nil {
		lpf.logPatternCountMap = make(map[string]int)
	}
	items := strings.Split(value, ",")
	for _, item := range items {
		val := strings.SplitN(item, LogPatternFlagSeparator, 2)
		if len(val) != 2 {
			return fmt.Errorf("invalid format of the flag value: %v", val)
		}
		countThreshold, err := strconv.Atoi(val[0])
		if err != nil || countThreshold == 0 {
			return fmt.Errorf("invalid format for the flag value: %v: %v", val, err)
		}
		pattern := val[1]
		if pattern == "" {
			return fmt.Errorf("invalid format for the flag value: %v: %v", val, err)
		}
		lpf.logPatternCountMap[pattern] = countThreshold
	}
	return nil
}

// Type implements the Type function for flag.Value interface
func (lpf *LogPatternFlag) Type() string {
	return "logPatternFlag"
}

// GetLogPatternCountMap returns the stored log count map
func (lpf *LogPatternFlag) GetLogPatternCountMap() map[string]int {
	return lpf.logPatternCountMap
}
