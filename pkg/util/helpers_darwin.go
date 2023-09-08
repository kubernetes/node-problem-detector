/*
Copyright 2023 The Kubernetes Authors All rights reserved.

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
	"time"

	"github.com/shirou/gopsutil/v3/host"
)

// GetUptimeDuration returns the time elapsed since last boot.
// For example: "cos 77-12293.0.0", "ubuntu 16.04.6 LTS (Xenial Xerus)".
func GetUptimeDuration() (time.Duration, error) {
	ut, err := host.Uptime()
	if err != nil {
		return 0, err
	}
	return time.Duration(ut), nil
}

// GetOSVersion retrieves the version of the current operating system.
// For example: "darwin 13.5"".
func GetOSVersion() (string, error) {
	platform, _, version, err := host.PlatformInformation()
	if err != nil {
		return "", err
	}

	return platform + " " + version, nil
}
