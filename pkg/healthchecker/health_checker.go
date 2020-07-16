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
	"strings"
	"time"

	"github.com/golang/glog"

	"k8s.io/node-problem-detector/cmd/healthchecker/options"
	"k8s.io/node-problem-detector/pkg/healthchecker/types"
)

type healthChecker struct {
	enableRepair    bool
	healthCheckFunc func() bool
	// The repair is "best-effort" and ignores the error from the underlying actions.
	// The bash commands to kill the process will fail if the service is down and hence ignore.
	repairFunc         func()
	uptimeFunc         func() (time.Duration, error)
	crictlPath         string
	healthCheckTimeout time.Duration
	coolDownTime       time.Duration
}

// NewHealthChecker returns a new health checker configured with the given options.
func NewHealthChecker(hco *options.HealthCheckerOptions) (types.HealthChecker, error) {
	hc := &healthChecker{
		enableRepair:       hco.EnableRepair,
		crictlPath:         hco.CriCtlPath,
		healthCheckTimeout: hco.HealthCheckTimeout,
		coolDownTime:       hco.CoolDownTime,
	}
	hc.healthCheckFunc = getHealthCheckFunc(hco)
	hc.repairFunc = getRepairFunc(hco)
	hc.uptimeFunc = getUptimeFunc(hco.SystemdService)
	return hc, nil
}

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
			execCommand(types.CmdTimeout, "systemctl", "kill", "--kill-who=main", hco.SystemdService)
		}
	default:
		// Just kill the service for all other components
		return func() {
			execCommand(types.CmdTimeout, "systemctl", "kill", "--kill-who=main", hco.SystemdService)
		}
	}
}

// getHealthCheckFunc returns the health check function based on the component.
func getHealthCheckFunc(hco *options.HealthCheckerOptions) func() bool {
	switch hco.Component {
	case types.KubeletComponent:
		return func() bool {
			httpClient := http.Client{Timeout: hco.HealthCheckTimeout}
			response, err := httpClient.Get(types.KubeletHealthCheckEndpoint)
			if err != nil || response.StatusCode != http.StatusOK {
				return false
			}
			return true
		}
	case types.DockerComponent:
		return func() bool {
			if _, err := execCommand(hco.HealthCheckTimeout, "docker", "ps"); err != nil {
				return false
			}
			return true
		}
	case types.CRIComponent:
		return func() bool {
			if _, err := execCommand(hco.HealthCheckTimeout, hco.CriCtlPath, "--runtime-endpoint="+hco.CriSocketPath, "--image-endpoint="+hco.CriSocketPath, "pods"); err != nil {
				return false
			}
			return true
		}
	}
	return nil
}

// CheckHealth checks for the health of the component and tries to repair if enabled.
// Returns true if healthy, false otherwise.
func (hc *healthChecker) CheckHealth() bool {
	healthy := hc.healthCheckFunc()
	if healthy {
		return true
	}
	// The service is unhealthy.
	// Attempt repair based on flag.
	if hc.enableRepair {
		glog.Infof("health-checker: component is unhealthy, proceeding to repair")
		// repair if the service has been up for the cool down period.
		uptime, err := hc.uptimeFunc()
		if err != nil {
			glog.Infof("health-checker: %v\n", err.Error())
		}
		glog.Infof("health-checker: component uptime: %v\n", uptime)
		if uptime > hc.coolDownTime {
			hc.repairFunc()
		}
	}
	return false
}

// execCommand executes the bash command and returns the (output, error) from command, error if timeout occurs.
func execCommand(timeout time.Duration, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	glog.Infof("health-checker: executing command : %v\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		glog.Infof("health-checker: command failed : %v, %v\n", err.Error(), out)
		return "", err
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}
