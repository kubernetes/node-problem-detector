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

package healthchecker

import (
	"context"
	"errors"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"

	"k8s.io/node-problem-detector/cmd/healthchecker/options"
	"k8s.io/node-problem-detector/pkg/healthchecker/types"
)

// getUptimeFunc returns the time for which the given service has been running.
func getUptimeFunc(service string) func() (time.Duration, error) {
	return func() (time.Duration, error) {
		// Using InactiveExitTimestamp to capture the exact time when systemd tried starting the service. The service will
		// transition from inactive -> activating and the timestamp is captured.
		// Source : https://www.freedesktop.org/wiki/Software/systemd/dbus/
		// Using ActiveEnterTimestamp resulted in race condition where the service was repeatedly killed by plugin when
		// RestartSec of systemd and invoke interval of plugin got in sync. The service was repeatedly killed in
		// activating state and hence ActiveEnterTimestamp was never updated.
		out, err := execCommand(types.CmdTimeout, "systemctl", "show", service, "--property=InactiveExitTimestamp")

		if err != nil {
			return time.Duration(0), err
		}
		val := strings.Split(out, "=")
		if len(val) < 2 {
			return time.Duration(0), errors.New("could not parse the service uptime time correctly")
		}
		t, err := time.Parse(types.UptimeTimeLayout, val[1])
		if err != nil {
			return time.Duration(0), err
		}
		return time.Since(t), nil
	}
}

// getRepairFunc returns the repair function based on the component.
func getRepairFunc(hco *options.HealthCheckerOptions) func() {
	switch hco.Component {
	case types.DockerComponent:
		// Use "docker ps" for docker health check. Not using crictl for docker to remove
		// dependency on the kubelet.
		return func() {
			execCommand(types.CmdTimeout, "pkill", "-SIGUSR1", "dockerd")
			execCommand(types.CmdTimeout, "systemctl", "kill", "--kill-who=main", hco.Service)
		}
	default:
		// Just kill the service for all other components
		return func() {
			execCommand(types.CmdTimeout, "systemctl", "kill", "--kill-who=main", hco.Service)
		}
	}
}

// getHealthCheckFunc returns the health check function based on the component.
func getHealthCheckFunc(hco *options.HealthCheckerOptions) func() (bool, error) {
	switch hco.Component {
	case types.KubeletComponent:
		return func() (bool, error) {
			httpClient := http.Client{Timeout: hco.HealthCheckTimeout}
			response, err := httpClient.Get(types.KubeletHealthCheckEndpoint)
			if err != nil || response.StatusCode != http.StatusOK {
				return false, nil
			}
			return true, nil
		}
	case types.DockerComponent:
		return func() (bool, error) {
			if _, err := execCommand(hco.HealthCheckTimeout, "docker", "ps"); err != nil {
				return false, nil
			}
			return true, nil
		}
	case types.CRIComponent:
		return func() (bool, error) {
			if _, err := execCommand(hco.HealthCheckTimeout, hco.CriCtlPath, "--runtime-endpoint="+hco.CriSocketPath, "--image-endpoint="+hco.CriSocketPath, "pods"); err != nil {
				return false, nil
			}
			return true, nil
		}
	}
	return nil
}

// execCommand executes the bash command and returns the (output, error) from command, error if timeout occurs.
func execCommand(timeout time.Duration, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	out, err := cmd.Output()
	if err != nil {
		glog.Infof("command %v failed: %v, %v\n", cmd, err, out)
		return "", err
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// checkForPattern returns (true, nil) if logPattern occurs less than logCountThreshold number of times since last
// service restart. (false, nil) otherwise.
func checkForPattern(service, logStartTime, logPattern string, logCountThreshold int) (bool, error) {
	out, err := execCommand(types.CmdTimeout, "/bin/sh", "-c",
		// Query service logs since the logStartTime
		`journalctl --unit "`+service+`" --since "`+logStartTime+
			// Grep the pattern
			`" | grep -i "`+logPattern+
			// Get the count of occurrences
			`" | wc -l`)
	if err != nil {
		return true, err
	}
	occurrences, err := strconv.Atoi(out)
	if err != nil {
		return true, err
	}
	if occurrences >= logCountThreshold {
		glog.Infof("%s failed log pattern check, %s occurrences: %v", service, logPattern, occurrences)
		return false, nil
	}
	return true, nil
}
