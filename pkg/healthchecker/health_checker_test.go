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
	"testing"
	"time"

	"k8s.io/node-problem-detector/pkg/healthchecker/types"
)

var repairCalled bool

func NewTestHealthChecker(repairFunc func(), healthCheckFunc func() bool, uptimeFunc func() (time.Duration, error), enableRepair bool) types.HealthChecker {
	repairCalled = false
	return &healthChecker{
		enableRepair:       enableRepair,
		healthCheckFunc:    healthCheckFunc,
		repairFunc:         repairFunc,
		uptimeFunc:         uptimeFunc,
		healthCheckTimeout: time.Second,
		coolDownTime:       2 * time.Second,
	}
}

func healthyFunc() bool {
	return true
}

func unhealthyFunc() bool {
	return false
}

func repairFunc() {
	repairCalled = true
}

func longServiceUptimeFunc() (time.Duration, error) {
	return 1 * time.Hour, nil
}

func shortServiceUptimeFunc() (time.Duration, error) {
	return 1 * time.Second, nil
}

func TestHealthCheck(t *testing.T) {
	for _, tc := range []struct {
		description     string
		enableRepair    bool
		healthy         bool
		healthCheckFunc func() bool
		uptimeFunc      func() (time.Duration, error)
		repairFunc      func()
		repairCalled    bool
	}{
		{
			description:     "healthy component",
			enableRepair:    true,
			healthy:         true,
			healthCheckFunc: healthyFunc,
			repairFunc:      repairFunc,
			uptimeFunc:      shortServiceUptimeFunc,
			repairCalled:    false,
		},
		{
			description:     "unhealthy component and disabled repair",
			enableRepair:    false,
			healthy:         false,
			healthCheckFunc: unhealthyFunc,
			repairFunc:      repairFunc,
			uptimeFunc:      shortServiceUptimeFunc,
			repairCalled:    false,
		},
		{
			description:     "unhealthy component, enabled repair and component in cool dowm",
			enableRepair:    true,
			healthy:         false,
			healthCheckFunc: unhealthyFunc,
			repairFunc:      repairFunc,
			uptimeFunc:      shortServiceUptimeFunc,
			repairCalled:    false,
		},
		{
			description:     "unhealthy component, enabled repair and component out of cool dowm",
			enableRepair:    true,
			healthy:         false,
			healthCheckFunc: unhealthyFunc,
			repairFunc:      repairFunc,
			uptimeFunc:      longServiceUptimeFunc,
			repairCalled:    true,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			hc := NewTestHealthChecker(tc.repairFunc, tc.healthCheckFunc, tc.uptimeFunc, tc.enableRepair)
			healthy := hc.CheckHealth()
			if healthy != tc.healthy {
				t.Errorf("incorrect health returned got %t; expected %t", healthy, tc.healthy)
			}
			if repairCalled != tc.repairCalled {
				t.Errorf("incorrect repairCalled got %t; expected %t", repairCalled, tc.repairCalled)
			}
		})
	}
}
