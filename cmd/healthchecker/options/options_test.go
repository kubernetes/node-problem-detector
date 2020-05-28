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

package options

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/healthchecker/types"
)

func TestIsValid(t *testing.T) {
	testCases := []struct {
		name        string
		hco         HealthCheckerOptions
		expectError bool
	}{
		{
			name: "valid component",
			hco: HealthCheckerOptions{
				Component: types.KubeletComponent,
			},
			expectError: false,
		},
		{
			name: "invalid component",
			hco: HealthCheckerOptions{
				Component: "wrongComponent",
			},
			expectError: true,
		},
		{
			name: "empty crictl-path with cri",
			hco: HealthCheckerOptions{
				Component:    types.CRIComponent,
				CriCtlPath:   "",
				EnableRepair: false,
			},
			expectError: true,
		},
		{
			name: "empty systemd-service and repair enabled",
			hco: HealthCheckerOptions{
				Component:      types.KubeletComponent,
				EnableRepair:   true,
				SystemdService: "",
			},
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.expectError {
				assert.Error(t, test.hco.IsValid(), "HealthChecker option %+v is invalid. Expected IsValid to return error.", test.hco)
			} else {
				assert.NoError(t, test.hco.IsValid(), "HealthChecker option %+v is valid. Expected IsValid to return nil.", test.hco)
			}
		})
	}
}
