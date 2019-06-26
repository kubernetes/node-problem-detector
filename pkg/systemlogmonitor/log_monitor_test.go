/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package systemlogmonitor

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemmetrics"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

const (
	testSource     = "TestSource"
	testConditionA = "TestConditionA"
	testConditionB = "TestConditionB"
)

func TestRegistration(t *testing.T) {
	assert.NotPanics(t,
		func() { problemdaemon.GetProblemDaemonHandlerOrDie("system-log-monitor") },
		"System log monitor failed to register itself as a problem daemon.")
}

func TestGenerateStatusForConditions(t *testing.T) {
	initConditions := []types.Condition{
		{
			Type:       testConditionA,
			Status:     types.True,
			Transition: time.Unix(500, 500),
			Reason:     "initial reason",
		},
		{
			Type:       testConditionB,
			Status:     types.False,
			Transition: time.Unix(500, 500),
		},
	}
	logs := []*logtypes.Log{
		{
			Timestamp: time.Unix(1000, 1000),
			Message:   "test message 1",
		},
		{
			Timestamp: time.Unix(2000, 2000),
			Message:   "test message 2",
		},
	}
	for c, test := range []struct {
		rule     logtypes.Rule
		expected types.Status
	}{
		// Do not need Pattern because we don't do pattern match in this test
		{
			rule: logtypes.Rule{
				Type:      types.Perm,
				Condition: testConditionA,
				Reason:    "test reason",
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{util.GenerateConditionChangeEvent(
					testConditionA,
					types.True,
					"test reason",
					time.Unix(1000, 1000),
				)},
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(1000, 1000),
						Reason:     "test reason",
						Message:    "test message 1\ntest message 2",
					},
					initConditions[1],
				},
			},
		},
		// Should not update transition time when status and reason are not changed.
		{
			rule: logtypes.Rule{
				Type:      types.Perm,
				Condition: testConditionA,
				Reason:    "initial reason",
			},
			expected: types.Status{
				Source: testSource,
				Conditions: []types.Condition{
					{
						Type:       testConditionA,
						Status:     types.True,
						Transition: time.Unix(500, 500),
						Reason:     "initial reason",
					},
					initConditions[1],
				},
			},
		},
		{
			rule: logtypes.Rule{
				Type:   types.Temp,
				Reason: "test reason",
			},
			expected: types.Status{
				Source: testSource,
				Events: []types.Event{{
					Severity:  types.Warn,
					Timestamp: time.Unix(1000, 1000),
					Reason:    "test reason",
					Message:   "test message 1\ntest message 2",
				}},
				Conditions: initConditions,
			},
		},
	} {
		l := &logMonitor{
			config: MonitorConfig{
				Source: testSource,
			},
			// Copy the init conditions to make sure it's not changed
			// during the test.
			conditions: append([]types.Condition{}, initConditions...),
		}
		(&l.config).ApplyDefaultConfiguration()
		got := l.generateStatus(logs, test.rule)
		if !reflect.DeepEqual(&test.expected, got) {
			t.Errorf("case %d: expected status %+v, got %+v", c+1, test.expected, got)
		}
	}
}

func TestGenerateStatusForMetrics(t *testing.T) {
	testCases := []struct {
		name            string
		conditions      []types.Condition
		triggeredRules  []logtypes.Rule
		expectedMetrics []metrics.Int64MetricRepresentation
	}{
		{
			name:            "one temporary problem that has not happened",
			conditions:      []types.Condition{},
			triggeredRules:  []logtypes.Rule{},
			expectedMetrics: []metrics.Int64MetricRepresentation{},
		},
		{
			name:       "one temporary problem happened once",
			conditions: []types.Condition{},
			triggeredRules: []logtypes.Rule{
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  1,
				},
			},
		},
		{
			name:       "one temporary problem happened twice",
			conditions: []types.Condition{},
			triggeredRules: []logtypes.Rule{
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  2,
				},
			},
		},
		{
			name:       "two different temporary problems happened",
			conditions: []types.Condition{},
			triggeredRules: []logtypes.Rule{
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
				{
					Type:   types.Temp,
					Reason: "problem reason bar",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  1,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason bar"},
					Value:  1,
				},
			},
		},
		{
			name: "one permanent problem that is happening",
			conditions: []types.Condition{
				{
					Type:   "ConditionA",
					Status: types.False,
				},
			},
			triggeredRules: []logtypes.Rule{
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  1,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  1,
				},
			},
		},
		{
			name: "one permanent problem observed twice with same reason",
			conditions: []types.Condition{
				{
					Type:   "ConditionA",
					Status: types.False,
				},
			},
			triggeredRules: []logtypes.Rule{
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  1,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  1,
				},
			},
		},
		{
			name: "one permanent problem observed twice with different reasons",
			conditions: []types.Condition{
				{
					Type:   "ConditionA",
					Status: types.False,
				},
			},
			triggeredRules: []logtypes.Rule{
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason bar",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason bar"},
					Value:  1,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  1,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason bar"},
					Value:  1,
				},
			},
		},
		{
			name: "two permanent problem observed once each",
			conditions: []types.Condition{
				{
					Type:   "ConditionA",
					Status: types.False,
				},
				{
					Type:   "ConditionB",
					Status: types.False,
				},
			},
			triggeredRules: []logtypes.Rule{
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionB",
					Reason:    "problem reason bar",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  1,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionB", "reason": "problem reason bar"},
					Value:  1,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  1,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason bar"},
					Value:  1,
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			l := &logMonitor{}
			l.conditions = test.conditions
			(&l.config).ApplyDefaultConfiguration()

			originalGlobalProblemMetricsManager := problemmetrics.GlobalProblemMetricsManager
			defer func() {
				problemmetrics.GlobalProblemMetricsManager = originalGlobalProblemMetricsManager
			}()

			fakePMM, fakeProblemCounter, fakeProblemGauge := problemmetrics.NewProblemMetricsManagerStub()
			problemmetrics.GlobalProblemMetricsManager = fakePMM

			for _, rule := range test.triggeredRules {
				l.generateStatus([]*logtypes.Log{{}}, rule)
			}

			gotMetrics := append(fakeProblemCounter.ListMetrics(), fakeProblemGauge.ListMetrics()...)

			assert.ElementsMatch(t, test.expectedMetrics, gotMetrics,
				"expected metrics: %+v, got: %+v", test.expectedMetrics, gotMetrics)
		})
	}
}

func TestInitializeProblemMetricsOrDie(t *testing.T) {
	testCases := []struct {
		name            string
		rules           []logtypes.Rule
		expectedMetrics []metrics.Int64MetricRepresentation
	}{
		{
			name:            "no problem type at all",
			rules:           []logtypes.Rule{},
			expectedMetrics: []metrics.Int64MetricRepresentation{},
		},
		{
			name: "one type of temporary problem",
			rules: []logtypes.Rule{
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  0,
				},
			},
		},
		{
			name: "one type of permanent problem",
			rules: []logtypes.Rule{
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  0,
				},
			},
		},
		{
			name: "duplicate temporary problem types",
			rules: []logtypes.Rule{
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  0,
				},
			},
		},
		{
			name: "multiple temporary problem types",
			rules: []logtypes.Rule{
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
				{
					Type:   types.Temp,
					Reason: "problem reason bar",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason bar"},
					Value:  0,
				},
			},
		},
		{
			name: "multiple permanent problem types with same condition",
			rules: []logtypes.Rule{
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason bar",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason bar"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason bar"},
					Value:  0,
				},
			},
		},
		{
			name: "multiple permanent problem types with different conditions",
			rules: []logtypes.Rule{
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionB",
					Reason:    "problem reason bar",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionB", "reason": "problem reason bar"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason bar"},
					Value:  0,
				},
			},
		},
		{
			name: "duplicate permanent problem types",
			rules: []logtypes.Rule{
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  0,
				},
			},
		},
		{
			name: "mixture of temporary and permanent problem types",
			rules: []logtypes.Rule{
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason hello",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionA",
					Reason:    "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionB",
					Reason:    "problem reason foo",
				},
				{
					Type:      types.Perm,
					Condition: "ConditionB",
					Reason:    "problem reason bar",
				},
				{
					Type:   types.Temp,
					Reason: "problem reason foo",
				},
				{
					Type:   types.Temp,
					Reason: "problem reason bar",
				},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason hello"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionA", "reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionB", "reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ConditionB", "reason": "problem reason bar"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason hello"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason foo"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "problem reason bar"},
					Value:  0,
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			l := &logMonitor{}
			(&l.config).ApplyDefaultConfiguration()

			originalGlobalProblemMetricsManager := problemmetrics.GlobalProblemMetricsManager
			defer func() {
				problemmetrics.GlobalProblemMetricsManager = originalGlobalProblemMetricsManager
			}()

			fakePMM, fakeProblemCounter, fakeProblemGauge := problemmetrics.NewProblemMetricsManagerStub()
			problemmetrics.GlobalProblemMetricsManager = fakePMM

			initializeProblemMetricsOrDie(test.rules)

			gotMetrics := append(fakeProblemCounter.ListMetrics(), fakeProblemGauge.ListMetrics()...)

			assert.ElementsMatch(t, test.expectedMetrics, gotMetrics,
				"expected metrics: %+v, got: %+v", test.expectedMetrics, gotMetrics)
		})
	}
}
