/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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
package util

import (
	"fmt"
	"time"

	"github.com/cobaugh/osrelease"

	"k8s.io/node-problem-detector/pkg/types"
)

var osReleasePath = "/etc/os-release"

// GenerateConditionChangeEvent generates an event for condition change.
func GenerateConditionChangeEvent(t string, status types.ConditionStatus, reason string, timestamp time.Time) types.Event {
	return types.Event{
		Severity:  types.Info,
		Timestamp: timestamp,
		Reason:    reason,
		Message:   fmt.Sprintf("Node condition %s is now: %s, reason: %s", t, status, reason),
	}
}

func GetStartTime(now time.Time, uptimeDuration time.Duration, lookbackStr string, delayStr string) (time.Time, error) {
	startTime := now.Add(-uptimeDuration)

	// Delay startTime if delay duration is set, so that the log watcher can skip
	// the logs in delay duration and wait until the node is stable.
	if delayStr != "" {
		delay, err := time.ParseDuration(delayStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse delay duration %q: %v", delayStr, err)
		}
		// Notice that when delay > uptime, startTime is actually after now, which is fine.
		startTime = startTime.Add(delay)
	}

	// Addjust startTime according to lookback duration
	lookbackStartTime := now
	if lookbackStr != "" {
		lookback, err := time.ParseDuration(lookbackStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse lookback duration %q: %v", lookbackStr, err)
		}
		lookbackStartTime = now.Add(-lookback)
	}
	if startTime.Before(lookbackStartTime) {
		startTime = lookbackStartTime
	}

	return startTime, nil
}

// GetOSVersion retrieves the version of the current operating system.
// For example: "cos 77-12293.0.0", "ubuntu 16.04.6 LTS (Xenial Xerus)".
func GetOSVersion() (string, error) {
	osReleaseMap, err := osrelease.ReadFile(osReleasePath)
	if err != nil {
		return "", err
	}
	switch osReleaseMap["ID"] {
	case "cos":
		return getCOSVersion(osReleaseMap), nil
	case "debian":
		return getDebianVersion(osReleaseMap), nil
	case "ubuntu":
		return getDebianVersion(osReleaseMap), nil
	case "centos":
		return getDebianVersion(osReleaseMap), nil
	case "rhel":
		return getDebianVersion(osReleaseMap), nil
	default:
		return "", fmt.Errorf("Unsupported ID in /etc/os-release: %q", osReleaseMap["ID"])
	}
}

func getCOSVersion(osReleaseMap map[string]string) string {
	// /etc/os-release syntax for COS is defined here:
	// https://chromium.git.corp.google.com/chromiumos/docs/+/8edec95a297edfd8f1290f0f03a8aa35795b516b/os_config.md
	return fmt.Sprintf("%s %s-%s", osReleaseMap["ID"], osReleaseMap["VERSION"], osReleaseMap["BUILD_ID"])
}

func getDebianVersion(osReleaseMap map[string]string) string {
	// /etc/os-release syntax for Debian is defined here:
	// https://manpages.debian.org/testing/systemd/os-release.5.en.html
	return fmt.Sprintf("%s %s", osReleaseMap["ID"], osReleaseMap["VERSION"])
}
