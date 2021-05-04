/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"

	"k8s.io/node-problem-detector/cmd/healthchecker/options"
	"k8s.io/node-problem-detector/pkg/healthchecker/types"
	"k8s.io/node-problem-detector/pkg/util"
)

// getUptimeFunc returns the time for which the given service has been running.
func getUptimeFunc(service string) func() (time.Duration, error) {
	return func() (time.Duration, error) {
		// Using the WinEvent Log Objects to find the Service logs' time when the Service last entered running state.
		// The powershell command formats the TimeCreated of the event log in RFC1123Pattern.
		// However, because the time library parser does not recognize the ',' in this RFC1123Pattern format,
		// it is manually removed before parsing it using the UptimeTimeLayout.
		getTimeCreatedCmd := "(Get-WinEvent -Logname System | Where-Object {$_.Message -Match '.*(" + service +
			").*(running).*'} | Select-Object -Property TimeCreated -First 1 | foreach {$_.TimeCreated.ToString('R')} | Out-String).Trim()"
		out, err := powershell(getTimeCreatedCmd)
		if err != nil {
			return time.Duration(0), err
		}
		if out == "" {
			return time.Duration(0), fmt.Errorf("service time creation not found for %s", service)
		}
		out = strings.ReplaceAll(out, ",", "")
		t, err := time.Parse(types.UptimeTimeLayout, out)
		if err != nil {
			return time.Duration(0), err
		}
		return time.Since(t), nil
	}
}

// getRepairFunc returns the repair function based on the component.
func getRepairFunc(hco *options.HealthCheckerOptions) func() {
	// Restart-Service will stop and attempt to start the service
	return func() {
		powershell("Restart-Service", hco.Service)
	}
}

// getHealthCheckFunc returns the health check function based on the component.
func getHealthCheckFunc(hco *options.HealthCheckerOptions) func() (bool, error) {
	switch hco.Component {
	case types.KubeletComponent:
		return healthCheckEndpointOKFunc(types.KubeletHealthCheckEndpoint, hco.HealthCheckTimeout)
	case types.KubeProxyComponent:
		return healthCheckEndpointOKFunc(types.KubeProxyHealthCheckEndpoint, hco.HealthCheckTimeout)
	case types.DockerComponent:
		return func() (bool, error) {
			if _, err := execCommand("docker.exe", "ps"); err != nil {
				return false, nil
			}
			return true, nil
		}
	case types.CRIComponent:
		return func() (bool, error) {
			if _, err := execCommand(hco.CriCtlPath, "--runtime-endpoint="+hco.CriSocketPath, "--image-endpoint="+hco.CriSocketPath, "pods"); err != nil {
				return false, nil
			}
			return true, nil
		}
	}
	return nil
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

// execCommand creates a new process, executes the command, and returns the (output, error) from command.
func execCommand(command string, args ...string) (string, error) {
	cmd := util.Exec(command, args...)
	return extractCommandOutput(cmd)
}

// powershell executes the arguments in powershell process and returns (output, error) from command.
func powershell(args ...string) (string, error) {
	cmd := util.Powershell(args...)
	return extractCommandOutput(cmd)
}

// Given an executable command, run and return the standard output, or error if command failed.
func extractCommandOutput(cmd *exec.Cmd) (string, error) {
	out, err := cmd.Output()
	if err != nil {
		glog.Infof("command %v failed: %v, %v\n", cmd, err, out)
		return "", err
	}
	return strings.TrimSuffix(string(out), "\r\n"), nil
}

// checkForPattern returns (true, nil) if logPattern occurs less than logCountThreshold number of times since last
// service restart. (false, nil) otherwise.
func checkForPattern(service, logStartTime, logPattern string, logCountThreshold int) (bool, error) {
	countPatternLogCmd := "@(Get-WinEvent -Logname System | Where-Object {($_.TimeCreated -ge ([datetime]::ParseExact('" + logStartTime +
		"','" + types.LogParsingTimeFormat + "', $null))) -and ($_.Message -Match '" + logPattern + "')}).count"

	out, err := powershell(countPatternLogCmd)
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
