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
	"time"

	"github.com/golang/glog"
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
	logPatternHealthy, err := logPatternHealthCheck(hc.service, hc.logPatternsToCheck)
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

// logPatternHealthCheck checks for the provided logPattern occurrences in the service logs.
// Returns true if the pattern is empty or does not exist logThresholdCount times since start of service, false otherwise.
func logPatternHealthCheck(service string, logPatternsToCheck map[string]int) (bool, error) {
	if len(logPatternsToCheck) == 0 {
		return true, nil
	}
	uptimeFunc := getUptimeFunc(service)
	uptime, err := uptimeFunc()
	if err != nil {
		return true, err
	}
	logStartTime := time.Now().Add(-uptime).Format(types.LogParsingTimeLayout)
	if err != nil {
		return true, err
	}
	for pattern, count := range logPatternsToCheck {
		healthy, err := checkForPattern(service, logStartTime, pattern, count)
		if err != nil || !healthy {
			return healthy, err
		}
	}
	return true, nil
}
