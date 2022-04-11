/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package custompluginmonitor

import (
	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/problemdaemon"
)

const (
	testSource     = "testSource"
	testConditionA = "TestConditionA"
	testConditionB = "TestConditionB"
)

var (
	defaultEnableMetricsReporting            = true
	defaultMessageChangeBasedConditionUpdate = true
)

func TestRegistration(t *testing.T) {
	assert.NotPanics(t,
		func() { problemdaemon.GetProblemDaemonHandlerOrDie("custom-plugin-monitor") },
		"Custom plugin monitor failed to register itself as a problem daemon.")
}

func TestGenerateStatusFromFalse(t *testing.T) {
	initConditions := []types.Condition{
		{
			Type:       testConditionA,
			Status:     types.False,
			Transition: time.Unix(500, 500),
			Reason:     "initial reason A",
		},
		{
			Type:       testConditionB,
			Status:     types.False,
			Transition: time.Unix(500, 500),
			Reason:     "initial reason B",
		},
	}

	for i, test := range []struct {
		resultArray []cpmtypes.Result
		expected    types.Status
	}{
		{
			// case 1: 2 results (1 NonOK/True, 1 OK/False) for testConditionA
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 1A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 2A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 1A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 1A",
						Message:    "test message 1A",
					},
					initConditions[1],
				},
			},
		},
		{
			// case 2: case 1 with different order
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 2A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 1A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 1A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 1A",
						Message:    "test message 1A",
					},
					initConditions[1],
				},
			},
		},
		{
			// case 3: 2 results (2 OK/False/Unkown) for testConditionA
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 1A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.Unknown,
					Message:    "test message 2A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.False,
						"initial reason A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: initConditions,
			},
		},
		{
			// case 4: 3 results (2 NonOK/True) for testConditionA. First result takes precedence
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 1A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 2A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 3A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 3A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 1A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 1A",
						Message:    "test message 1A",
					},
					initConditions[1],
				},
			},
		},
		{
			// case 5: 2 results (2 NonOK/True) for both conditions
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 1A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionB,
						Reason:    "test reason 1B",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 1B",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 1A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.True,
						"test reason 1B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 1A",
						Message:    "test message 1A",
					},
					{
						Type:       testConditionB,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 1B",
						Message:    "test message 1B",
					},
				},
			},
		},
	} {
		c := &customPluginMonitor{
			config: cpmtypes.CustomPluginConfig{
				Source:            testSource,
				DefaultConditions: initConditions,
			},
			conditions: initialConditions(initConditions),
		}

		c.config.EnableMetricsReporting = &defaultEnableMetricsReporting

		actual := c.generateStatus(test.resultArray)
		resetTestTimestamp(actual)

		if !reflect.DeepEqual(&test.expected, actual) {
			t.Errorf("case %d: expected status %+v, got %+v", i+1, test.expected, actual)
		}
	}
}

func TestGenerateStatusFromTrue(t *testing.T) {
	defaultConditions := []types.Condition{
		{
			Type:       testConditionA,
			Status:     types.False,
			Transition: time.Unix(500, 500),
			Reason:     "initial reason A",
		},
		{
			Type:       testConditionB,
			Status:     types.False,
			Transition: time.Unix(500, 500),
			Reason:     "initial reason B",
		},
	}

	for i, test := range []struct {
		resultArray []cpmtypes.Result
		expected    types.Status
	}{
		{
			// case 1: 2 results (1 NonOK/True, 1 OK/False) for testConditionA. Condition reason don't match
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 3A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 3A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 2A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 1A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: initTestConditions(),
			},
		},
		{
			// case 2: case 1 with different order
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 2A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 3A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 3A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 1A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: initTestConditions(),
			},
		},
		{
			// case 3: 2 results (1 NonOK/True, 1 OK/False) for testConditionA. Condition reason match, reason switch.
			// shouldn't flap (testConditionA set to OK/False, then set to NonOK/True by reason 2A)
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 1A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 2A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 2A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 2A",
						Message:    "test message 2A",
					},
					initTestConditions()[1],
				},
			},
		},
		{
			// case 4: case 3 with different order. test reason 2A shouldn't be ignored
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 2A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 1A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 2A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 2A",
						Message:    "test message 2A",
					},
					initTestConditions()[1],
				},
			},
		},
		{
			// case 5: 2 results (2 OK/False/Unkown) for testConditionA.
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 2A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.Unknown,
					Message:    "test message 1A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.Unknown,
						"initial reason A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.Unknown,
						Transition: time.Unix(500, 500),
						Reason:     "initial reason A",
						Message:    "test message 1A",
					},
					initTestConditions()[1],
				},
			},
		},
		{
			// case 6: 3 results (2 NonOK/True) for testConditionA.
			// Condition reason match, reason switch to result that came first
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.OK,
					Message:    "test message 1A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 2A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 2A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 3A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 3A",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 2A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.False,
						"initial reason B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 2A",
						Message:    "test message 2A",
					},
					initTestConditions()[1],
				},
			},
		},
		{
			// case 7: 2 results (2 NonOk/True) for both conditions.
			resultArray: []cpmtypes.Result{
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionA,
						Reason:    "test reason 1A",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 1A",
				},
				{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: testConditionB,
						Reason:    "test reason 1B",
					},
					ExitStatus: cpmtypes.NonOK,
					Message:    "test message 1B",
				},
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{
					util.GenerateConditionChangeEvent(
						testConditionA,
						types.True,
						"test reason 1A",
						time.Unix(500, 500),
					),
					util.GenerateConditionChangeEvent(
						testConditionB,
						types.True,
						"test reason 1B",
						time.Unix(500, 500),
					),
				},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 1A",
						Message:    "test message 1A",
					},
					{
						Type:       testConditionB,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "test reason 1B",
						Message:    "test message 1B",
					},
				},
			},
		},
	} {
		c := &customPluginMonitor{
			config: cpmtypes.CustomPluginConfig{
				Source:            testSource,
				DefaultConditions: defaultConditions,
			},
			conditions: initTestConditions(),
		}

		c.config.EnableMetricsReporting = &defaultEnableMetricsReporting
		c.config.PluginGlobalConfig.EnableMessageChangeBasedConditionUpdate = &defaultMessageChangeBasedConditionUpdate

		actual := c.generateStatus(test.resultArray)
		resetTestTimestamp(actual)

		if !reflect.DeepEqual(&test.expected, actual) {
			t.Errorf("case %d: expected status %+v, got %+v", i+1, test.expected, actual)
		}
	}
}

func resetTestTimestamp(status *types.Status) {
	for i := range status.Events {
		status.Events[i].Timestamp = time.Unix(500, 500)
	}

	for i := range status.Conditions {
		status.Conditions[i].Transition = time.Unix(500, 500)
	}
}

func initTestConditions() []types.Condition {
	return []types.Condition{
		{
			Type:       testConditionA,
			Status:     types.True,
			Transition: time.Unix(500, 500),
			Reason:     "test reason 1A",
		},
		{
			Type:       testConditionB,
			Status:     types.False,
			Transition: time.Unix(500, 500),
			Reason:     "initial reason B",
		},
	}
}
