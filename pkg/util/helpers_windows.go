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

	"github.com/shirou/gopsutil/v3/host"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// GetUptimeDuration returns the time elapsed since last boot.
func GetUptimeDuration() (time.Duration, error) {
	ut, err := host.Uptime()
	if err != nil {
		return 0, err
	}
	return time.Duration(ut), nil
}

// GetOSVersion retrieves the version of the current operating system.
// For example: "windows 10.0.17763.1697 (Windows Server 2016 Datacenter)".
func GetOSVersion() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	productName, _, err := k.GetStringValue("ProductName")
	if err != nil {
		productName = "windows"
	}

	ubr, _, err := k.GetIntegerValue("UBR")
	if err != nil {
		ubr = 0
	}

	major, minor, build := windows.RtlGetNtVersionNumbers()

	return fmt.Sprintf("windows %d.%d.%d.%d (%s)", major, minor, build, ubr, productName), nil
}
