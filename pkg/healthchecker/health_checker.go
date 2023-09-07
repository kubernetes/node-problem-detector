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
	"net/http"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"
	"k8s.io/node-problem-detector/cmd/healthchecker/options"
	"k8s.io/node-problem-detector/pkg/healthchecker/types"
)

type healthChecker struct {
	component       string
	service         string
	enableRepair    bool
	healthCheckFunc func() (bool, error)
	// The repair is "best-effort" and ignores the error from the underlying actions.
	// The bash commands to kill the process will fail if the service is down and hence ignore.
	repairFunc         func()
	uptimeFunc         func() (time.Duration, error)
	crictlPath         string
	healthCheckTimeout time.Duration
	coolDownTime       time.Duration
	loopBackTime       time.Duration
	logPatternsToCheck map[string]int
}

// NewHealthChecker returns a new health checker configured with the given options.
func NewHealthChecker(hco *options.HealthCheckerOptions) (types.HealthChecker, error) {
	hc := &healthChecker{
		component:          hco.Component,
		enableRepair:       hco.EnableRepair,
		crictlPath:         hco.CriCtlPath,
		healthCheckTimeout: hco.HealthCheckTimeout,
		coolDownTime:       hco.CoolDownTime,
		service:            hco.Service,
		loopBackTime:       hco.LoopBackTime,
		logPatternsToCheck: hco.LogPatterns.GetLogPatternCountMap(),
	}
	hc.healthCheckFunc = getHealthCheckFunc(hco)
	hc.repairFunc = getRepairFunc(hco)
	hc.uptimeFunc = getUptimeFunc(hco.Service)
	return hc, nil
}

// CheckHealth checks for the health of the component and tries to repair if enabled.
// Returns true if healthy, false otherwise.
func (hc *healthChecker) CheckHealth() (bool, error) {
	healthy, err := hc.healthCheckFunc()
	if err != nil {
		return healthy, err
	}
	logPatternHealthy, err := logPatternHealthCheck(hc.service, hc.loopBackTime, hc.logPatternsToCheck)
	if err != nil {
		return logPatternHealthy, err
	}
	if healthy && logPatternHealthy {
		return true, nil
	}

	// The service is unhealthy.
	// Attempt repair based on flag.
	if hc.enableRepair {
		// repair if the service has been up for the cool down period.
		uptime, err := hc.uptimeFunc()
		if err != nil {
			klog.Infof("error in getting uptime for %v: %v\n", hc.component, err)
			return false, nil
		}
		klog.Infof("%v is unhealthy, component uptime: %v\n", hc.component, uptime)
		if uptime > hc.coolDownTime {
			klog.Infof("%v cooldown period of %v exceeded, repairing", hc.component, hc.coolDownTime)
			hc.repairFunc()
		}
	}
	return false, nil
}

// logPatternHealthCheck checks for the provided logPattern occurrences in the service logs.
// Returns true if the pattern is empty or does not exist logThresholdCount times since start of service, false otherwise.
func logPatternHealthCheck(service string, loopBackTime time.Duration, logPatternsToCheck map[string]int) (bool, error) {
	if len(logPatternsToCheck) == 0 {
		return true, nil
	}
	uptimeFunc := getUptimeFunc(service)
	klog.Infof("Getting uptime for service: %v\n", service)
	uptime, err := uptimeFunc()
	if err != nil {
		klog.Warningf("Failed to get the uptime: %+v", err)
		return true, err
	}

	logStartTime := time.Now().Add(-uptime).Format(types.LogParsingTimeLayout)
	if loopBackTime > 0 && uptime > loopBackTime {
		logStartTime = time.Now().Add(-loopBackTime).Format(types.LogParsingTimeLayout)
	}
	for pattern, count := range logPatternsToCheck {
		healthy, err := checkForPattern(service, logStartTime, pattern, count)
		if err != nil || !healthy {
			return healthy, err
		}
	}
	return true, nil
}

// healthCheckEndpointOKFunc returns a function to check the status of an http endpoint
func healthCheckEndpointOKFunc(endpoint string, timeout time.Duration) func() (bool, error) {
	return func() (bool, error) {
		httpClient := http.Client{Timeout: timeout}
		response, err := httpClient.Get(endpoint)
		if err != nil || response.StatusCode != http.StatusOK {
			return false, nil
		}
		return true, nil
	}
}

// getHealthCheckFunc returns the health check function based on the component.
func getHealthCheckFunc(hco *options.HealthCheckerOptions) func() (bool, error) {
	switch hco.Component {
	case types.KubeletComponent:
		return healthCheckEndpointOKFunc(types.KubeletHealthCheckEndpoint(), hco.HealthCheckTimeout)
	case types.KubeProxyComponent:
		return healthCheckEndpointOKFunc(types.KubeProxyHealthCheckEndpoint(), hco.HealthCheckTimeout)
	case types.DockerComponent:
		return func() (bool, error) {
			if _, err := execCommand(hco.HealthCheckTimeout, getDockerPath(), "ps"); err != nil {
				return false, nil
			}
			return true, nil
		}
	case types.CRIComponent:
		return func() (bool, error) {
			_, err := execCommand(
				hco.HealthCheckTimeout,
				hco.CriCtlPath,
				"--timeout="+hco.CriTimeout.String(),
				"--runtime-endpoint="+hco.CriSocketPath,
				"pods",
				"--latest",
			)
			if err != nil {
				return false, nil
			}
			return true, nil
		}
	default:
		klog.Warningf("Unsupported component: %v", hco.Component)
	}

	return nil
}

// execCommand executes the bash command and returns the (output, error) from command, error if timeout occurs.
func execCommand(timeout time.Duration, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Infof("command %v failed: %v, %v\n", cmd, err, out)
		return "", err
	}

	return strings.TrimSuffix(string(out), "\n"), nil
}
