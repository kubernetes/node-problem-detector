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

type healthChecker struct {
	component       string
	enableRepair    bool
	healthCheckFunc func() (bool, error)
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
		component:          hco.Component,
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
func getHealthCheckFunc(hco *options.HealthCheckerOptions) func() (bool, error) {
	switch hco.Component {
	case types.KubeletComponent:
		return getKubeletHealthCheckFunc(hco.HealthCheckTimeout)
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

// CheckHealth checks for the health of the component and tries to repair if enabled.
// Returns true if healthy, false otherwise.
func (hc *healthChecker) CheckHealth() (bool, error) {
	healthy, err := hc.healthCheckFunc()
	if err != nil {
		return healthy, err
	}
	if healthy {
		return true, nil
	}
	// The service is unhealthy.
	// Attempt repair based on flag.
	if hc.enableRepair {
		// repair if the service has been up for the cool down period.
		uptime, err := hc.uptimeFunc()
		if err != nil {
			glog.Infof("error in getting uptime for %v: %v\n", hc.component, err)
		}
		glog.Infof("%v is unhealthy, component uptime: %v\n", hc.component, uptime)
		if uptime > hc.coolDownTime {
			glog.Infof("%v cooldown period of %v exceeded, repairing", hc.component, hc.coolDownTime)
			hc.repairFunc()
		}
	}
	return false, nil
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

// kubeletHttpHealthCheck checks the health api response on kubelet.
// Returns true for healthy, false otherwise.
func kubeletHttpHealthCheck(healthCheckTimeout time.Duration) bool {
	httpClient := http.Client{Timeout: healthCheckTimeout}
	response, err := httpClient.Get(types.KubeletHealthCheckEndpoint)
	if err != nil || response.StatusCode != http.StatusOK {
		glog.Info("kubelet failed http health check")
		return false
	}
	return true
}

// kubeletConnectionHealthCheck checks for the kubelet-apiserver connection issue
// by checking repeated occurrences of log "use of closed network connection" in kubelet logs.
// Returns true if the pattern does not exist 10 times since start of service or the last 10 min, false otherwise.
func kubeletConnectionHealthCheck() (bool, error) {
	kubeletUptimeFunc := getUptimeFunc(types.KubeletComponent)
	uptime, err := kubeletUptimeFunc()
	if err != nil {
		return true, err
	}
	logStartTime := time.Now().Add(-uptime).Format(types.LogParsingTimeLayout)
	if err != nil {
		return true, err
	}
	out, err := execCommand(types.CmdTimeout, "/bin/sh", "-c",
		// Query kubelet logs since the logStartTime
		`journalctl --unit kubelet --since "`+logStartTime+
			// Grep the pattern for lost connection
			`" | grep -i "`+types.KubeletClosedConnectionLogPattern+
			// Get the count of occurrences
			`" | wc -l`)
	if err != nil {
		return true, err
	}
	occurrences, err := strconv.Atoi(out)
	if err != nil {
		return true, err
	}
	if occurrences >= types.KubeletClosedConnectionLogPatternThresholdCount {
		glog.Infof("kubelet failed apiserver connection check, log pattern occurrences: %v", occurrences)
		return false, nil
	}
	return true, nil
}

// getKubeletHealthCheckFunc returns a function that checks for kubelet health and
// return false if identified as unhealthy, true otherwise.
func getKubeletHealthCheckFunc(healthCheckTimeout time.Duration) func() (bool, error) {
	return func() (bool, error) {
		httpHealthy := kubeletHttpHealthCheck(healthCheckTimeout)
		connectionHealthy, err := kubeletConnectionHealthCheck()
		// The plugin will return Unknown status code in case there is any error in
		// checking kubelet health.
		if err != nil {
			glog.Infof("Error in determining apiserver connection health: %v", err)
			return false, err
		}
		healthy := httpHealthy && connectionHealthy
		return healthy, nil
	}
}
