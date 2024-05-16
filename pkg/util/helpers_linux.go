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
	"syscall"
	"time"

	"github.com/acobaugh/osrelease"
)

const (
	osReleasePath = "/etc/os-release"
)

// GetUptimeDuration returns the time elapsed since last boot.
func GetUptimeDuration() (time.Duration, error) {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0, fmt.Errorf("failed to get system info: %v", err)
	}
	return time.Duration(info.Uptime) * time.Second, nil
}

// GetOSVersion retrieves the version of the current operating system.
// For example: "cos 77-12293.0.0", "ubuntu 16.04.6 LTS (Xenial Xerus)".
func GetOSVersion() (string, error) {
	return getOSVersion(osReleasePath)
}

func getOSVersion(osReleasePath string) (string, error) {
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
	case "rocky":
		return getDebianVersion(osReleaseMap), nil
	case "rhel":
		return getDebianVersion(osReleaseMap), nil
	case "ol":
		return getDebianVersion(osReleaseMap), nil
	case "amzn":
		return getDebianVersion(osReleaseMap), nil
	case "sles":
		return getDebianVersion(osReleaseMap), nil
	case "mariner":
		return getDebianVersion(osReleaseMap), nil
	case "azurelinux":
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
