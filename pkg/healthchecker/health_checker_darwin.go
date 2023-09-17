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

package healthchecker

import (
	"runtime"
	"time"

	"k8s.io/klog/v2"
	"k8s.io/node-problem-detector/cmd/healthchecker/options"
)

// getUptimeFunc returns the time for which the given service has been running.
func getUptimeFunc(service string) func() (time.Duration, error) {
	klog.Fatalf("getUptimeFunc is not supported in %s", runtime.GOOS)
	return func() (time.Duration, error) { return time.Second, nil }
}

// getRepairFunc returns the repair function based on the component.
func getRepairFunc(hco *options.HealthCheckerOptions) func() {
	klog.Fatalf("getRepairFunc is not supported in %s", runtime.GOOS)
	return func() {}
}

// checkForPattern returns (true, nil) if logPattern occurs less than logCountThreshold number of times since last
// service restart. (false, nil) otherwise.
func checkForPattern(service, logStartTime, logPattern string, logCountThreshold int) (bool, error) {
	klog.Fatalf("checkForPattern is not supported in %s", runtime.GOOS)
	return false, nil
}

func getDockerPath() string {
	klog.Fatalf("getDockerPath is not supported in %s", runtime.GOOS)
	return ""
}
