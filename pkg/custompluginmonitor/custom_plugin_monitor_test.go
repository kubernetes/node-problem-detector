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
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/node-problem-detector/pkg/custompluginmonitor/plugin"
	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemmetrics"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/metrics"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

const (
	testSource         = "cpm-test"
	testConfigPath     = "test-config-path"
	testCondition      = "TestCondition"
	testConditionOK    = "TestConditionOK"
	testConditionOKMsg = "test condition is ok"
	otherCondition     = "OtherCondition"
	otherConditionOK   = "OtherConditionOK"
	otherOKMsg         = "other condition is ok"
	testProblemReason  = "TestConditionProblem"
	testTempReason     = "TestTempProblem"
	testWait           = 3 * time.Second
	settleWait         = 200 * time.Millisecond
)

// testEpoch is a fixed past instant stamped into Transition before a test runs.
var testEpoch = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

func TestRegistration(t *testing.T) {
	assert.NotPanics(t,
		func() { problemdaemon.GetProblemDaemonHandlerOrDie("custom-plugin-monitor") },
		"Custom plugin monitor failed to register itself as a problem daemon.")
}

// testOptions holds fields the tests and structs below may need.
type testOptions struct {
	defaultConditions      []types.Condition
	rules                  []*cpmtypes.CustomRule
	messageChange          bool
	enableMetricsReporting bool
	skipInitialStatus      bool
}

// defaultTestConditions returns two conditions. The condition under test is second.
func defaultTestConditions() []types.Condition {
	return []types.Condition{
		{Type: otherCondition, Reason: otherConditionOK, Message: otherOKMsg},
		{Type: testCondition, Reason: testConditionOK, Message: testConditionOKMsg},
	}
}

// newTestConfig builds a config with the defaults applied.
func newTestConfig(t *testing.T, o testOptions) cpmtypes.CustomPluginConfig {
	t.Helper()
	cfg := cpmtypes.CustomPluginConfig{
		Plugin:            "custom",
		Source:            testSource,
		DefaultConditions: o.defaultConditions,
		Rules:             o.rules,
	}
	require.NoError(t, (&cfg).ApplyConfiguration())
	messageChange := o.messageChange
	metricsReporting := o.enableMetricsReporting
	skipInitialStatus := o.skipInitialStatus
	cfg.PluginGlobalConfig.EnableMessageChangeBasedConditionUpdate = &messageChange
	cfg.PluginGlobalConfig.SkipInitialStatus = &skipInitialStatus
	cfg.EnableMetricsReporting = &metricsReporting
	return cfg
}

// newTestMonitor builds a monitor with its conditions initialized.
func newTestMonitor(t *testing.T, o testOptions) *customPluginMonitor {
	t.Helper()
	cfg := newTestConfig(t, o)
	c := &customPluginMonitor{
		configPath: testConfigPath,
		config:     cfg,
		plugin:     plugin.NewPlugin(cfg),
		statusChan: make(chan *types.Status, 1000),
		tomb:       tomb.NewTomb(),
	}
	c.initializeConditions()
	for i := range c.conditions {
		c.conditions[i].Transition = testEpoch
	}
	return c
}

// testPluginScript returns the path of a script in the plugin package's test data.
func testPluginScript(name string) string {
	ext := "sh"
	if runtime.GOOS == "windows" {
		ext = "cmd"
	}
	return filepath.Join("plugin", "test-data", name+"."+ext)
}

// permResult builds a permanent-rule result for the condition under test.
func permResult(status cpmtypes.Status, reason, message string) cpmtypes.Result {
	return cpmtypes.Result{
		Rule:       &cpmtypes.CustomRule{Type: types.Perm, Condition: testCondition, Reason: reason},
		ExitStatus: status,
		Message:    message,
	}
}

// stubProblemMetrics replaces the global metrics manager with a stub. Do not call t.Parallel.
func stubProblemMetrics(t *testing.T) (*metrics.FakeInt64Metric, *metrics.FakeInt64Metric) {
	t.Helper()
	original := problemmetrics.GlobalProblemMetricsManager
	t.Cleanup(func() { problemmetrics.GlobalProblemMetricsManager = original })
	pmm, counter, gauge := problemmetrics.NewProblemMetricsManagerStub()
	problemmetrics.GlobalProblemMetricsManager = pmm
	return counter, gauge
}

// receiveStatus returns the next status from the channel or fails the test after testWait.
func receiveStatus(t *testing.T, ch <-chan *types.Status) *types.Status {
	t.Helper()
	select {
	case status := <-ch:
		return status
	case <-time.After(testWait):
		t.Fatal("timed out waiting for a status")
		return nil
	}
}

// requireNoStatus fails the test if a status arrives within settleWait.
func requireNoStatus(t *testing.T, ch <-chan *types.Status) {
	t.Helper()
	select {
	case status := <-ch:
		t.Fatalf("expected no status, got %+v", status)
	case <-time.After(settleWait):
	}
}

// requireStop calls Stop and fails the test if Stop does not return within testWait.
func requireStop(t *testing.T, c *customPluginMonitor) {
	t.Helper()
	stopped := make(chan struct{})
	go func() {
		c.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(testWait):
		t.Fatal("Stop did not return")
	}
}

// step is one generateStatus call plus everything expected right after it.
type step struct {
	exitStatus  cpmtypes.Status
	ruleReason  string
	message     string
	wantStatus  types.ConditionStatus
	wantReason  string
	wantMessage string
	wantEvent   bool
}

func TestGenerateStatusForConditions(t *testing.T) {
	testCases := []struct {
		name          string
		messageChange bool
		ruleCondition string
		steps         []step
	}{
		{
			name:          "stays false, no event",
			ruleCondition: testCondition,
			steps: []step{
				{
					exitStatus: cpmtypes.OK, ruleReason: testProblemReason, message: "still fine",
					wantStatus: types.False, wantReason: testConditionOK, wantMessage: testConditionOKMsg, wantEvent: false,
				},
				{
					exitStatus: cpmtypes.OK, ruleReason: testProblemReason, message: "still fine",
					wantStatus: types.False, wantReason: testConditionOK, wantMessage: testConditionOKMsg, wantEvent: false,
				},
			},
		},
		{
			name:          "false to true then back to false restores the default",
			ruleCondition: testCondition,
			steps: []step{
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "boom",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "boom", wantEvent: true,
				},
				{
					exitStatus: cpmtypes.OK, ruleReason: testProblemReason, message: "all good",
					wantStatus: types.False, wantReason: testConditionOK, wantMessage: testConditionOKMsg, wantEvent: true,
				},
			},
		},
		{
			name:          "true to unknown keeps the result message",
			ruleCondition: testCondition,
			steps: []step{
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "boom",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "boom", wantEvent: true,
				},
				{
					exitStatus: cpmtypes.Unknown, ruleReason: testProblemReason, message: "plugin timed out",
					wantStatus: types.Unknown, wantReason: testConditionOK, wantMessage: "plugin timed out", wantEvent: true,
				},
			},
		},
		{
			name:          "unknown transitions",
			ruleCondition: testCondition,
			steps: []step{
				{
					exitStatus: cpmtypes.Unknown, ruleReason: testProblemReason, message: "cannot exec",
					wantStatus: types.Unknown, wantReason: testConditionOK, wantMessage: "cannot exec", wantEvent: true,
				},
				{
					exitStatus: cpmtypes.Unknown, ruleReason: testProblemReason, message: "cannot exec either",
					wantStatus: types.Unknown, wantReason: testConditionOK, wantMessage: "cannot exec", wantEvent: false,
				},
				{
					exitStatus: cpmtypes.OK, ruleReason: testProblemReason, message: "all good",
					wantStatus: types.False, wantReason: testConditionOK, wantMessage: testConditionOKMsg, wantEvent: true,
				},
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "boom",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "boom", wantEvent: true,
				},
			},
		},
		{
			name:          "reason change while true updates",
			ruleCondition: testCondition,
			steps: []step{
				{
					exitStatus: cpmtypes.NonOK, ruleReason: "ReasonA", message: "m",
					wantStatus: types.True, wantReason: "ReasonA", wantMessage: "m", wantEvent: true,
				},
				{
					exitStatus: cpmtypes.NonOK, ruleReason: "ReasonB", message: "m",
					wantStatus: types.True, wantReason: "ReasonB", wantMessage: "m", wantEvent: true,
				},
			},
		},
		{
			name:          "message change ignored when the flag is off",
			messageChange: false,
			ruleCondition: testCondition,
			steps: []step{
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "first",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "first", wantEvent: true,
				},
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "second",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "first", wantEvent: false,
				},
			},
		},
		{
			name:          "message change applied when the flag is on",
			messageChange: true,
			ruleCondition: testCondition,
			steps: []step{
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "first",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "first", wantEvent: true,
				},
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "second",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "second", wantEvent: true,
				},
			},
		},
		{
			name:          "identical result while true does not update",
			messageChange: true,
			ruleCondition: testCondition,
			steps: []step{
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "same",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "same", wantEvent: true,
				},
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "same",
					wantStatus: types.True, wantReason: testProblemReason, wantMessage: "same", wantEvent: false,
				},
			},
		},
		{
			name:          "rule for an unknown condition changes nothing",
			ruleCondition: "NoSuchCondition",
			steps: []step{
				{
					exitStatus: cpmtypes.NonOK, ruleReason: testProblemReason, message: "boom",
					wantStatus: types.False, wantReason: testConditionOK, wantMessage: testConditionOKMsg, wantEvent: false,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			c := newTestMonitor(t, testOptions{
				defaultConditions: defaultTestConditions(),
				messageChange:     test.messageChange,
			})
			for i, s := range test.steps {
				result := cpmtypes.Result{
					Rule: &cpmtypes.CustomRule{
						Type:      types.Perm,
						Condition: test.ruleCondition,
						Reason:    s.ruleReason,
					},
					ExitStatus: s.exitStatus,
					Message:    s.message,
				}
				got := c.generateStatus(result)

				// status.Conditions is an alias for c.conditions.
				require.Len(t, got.Conditions, 2, "step %d", i)
				assert.Equal(t, testSource, got.Source, "step %d", i)

				cond := got.Conditions[1]
				assert.Equal(t, testCondition, cond.Type, "step %d", i)
				assert.Equal(t, s.wantStatus, cond.Status, "step %d", i)
				assert.Equal(t, s.wantReason, cond.Reason, "step %d", i)
				assert.Equal(t, s.wantMessage, cond.Message, "step %d", i)

				// The rule does not target the first condition, so it must not change.
				other := got.Conditions[0]
				assert.Equal(t, types.False, other.Status, "step %d", i)
				assert.Equal(t, otherConditionOK, other.Reason, "step %d", i)
				assert.Equal(t, otherOKMsg, other.Message, "step %d", i)
				assert.True(t, other.Transition.Equal(testEpoch), "step %d", i)

				if !s.wantEvent {
					assert.Empty(t, got.Events, "step %d", i)
					continue
				}
				require.Len(t, got.Events, 1, "step %d", i)
				event := got.Events[0]
				// Assert severity literally; the want below reuses the production helper.
				wantSeverity := types.Info
				if s.wantStatus == types.True {
					wantSeverity = types.Warn
				}
				assert.Equal(t, wantSeverity, event.Severity, "step %d", i)
				// Feed the observed timestamp back so no clock value is asserted.
				want := util.GenerateConditionChangeEvent(
					testCondition, s.wantStatus, s.wantReason, s.wantMessage, event.Timestamp)
				assert.Equal(t, want, event, "step %d", i)
				assert.True(t, cond.Transition.Equal(event.Timestamp), "step %d", i)
				assert.False(t, cond.Transition.Equal(testEpoch), "step %d", i)
			}
		})
	}
}

func TestGenerateStatusForTemporaryProblem(t *testing.T) {
	testCases := []struct {
		name       string
		exitStatus cpmtypes.Status
		wantEvent  bool
	}{
		{name: "ok produces no event", exitStatus: cpmtypes.OK, wantEvent: false},
		{name: "non-ok produces an event", exitStatus: cpmtypes.NonOK, wantEvent: true},
		{name: "unknown produces an event", exitStatus: cpmtypes.Unknown, wantEvent: true},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			c := newTestMonitor(t, testOptions{defaultConditions: defaultTestConditions()})
			result := cpmtypes.Result{
				// Condition is set on purpose. A temporary rule must ignore it.
				Rule:       &cpmtypes.CustomRule{Type: types.Temp, Condition: testCondition, Reason: testTempReason},
				ExitStatus: test.exitStatus,
				Message:    "temporary problem detail",
			}
			got := c.generateStatus(result)

			assert.Equal(t, testSource, got.Source)
			if !test.wantEvent {
				assert.Empty(t, got.Events)
			} else {
				require.Len(t, got.Events, 1)
				event := got.Events[0]
				assert.Equal(t, types.Warn, event.Severity)
				assert.Equal(t, testTempReason, event.Reason)
				// A temporary event carries the raw plugin output.
				assert.Equal(t, "temporary problem detail", event.Message)
				assert.False(t, event.Timestamp.IsZero())
			}
			// No condition may move.
			require.Len(t, got.Conditions, 2)
			for _, cond := range got.Conditions {
				assert.Equal(t, types.False, cond.Status)
				assert.True(t, cond.Transition.Equal(testEpoch))
			}
		})
	}
}

func TestGenerateStatusMetrics(t *testing.T) {
	testCases := []struct {
		name                   string
		enableMetricsReporting bool
		results                []cpmtypes.Result
		expectedMetrics        []metrics.Int64MetricRepresentation
	}{
		{
			name:                   "reporting disabled records nothing",
			enableMetricsReporting: false,
			results:                []cpmtypes.Result{permResult(cpmtypes.NonOK, testProblemReason, "boom")},
			expectedMetrics:        []metrics.Int64MetricRepresentation{},
		},
		{
			name:                   "problem detected increments the counter and raises the gauge",
			enableMetricsReporting: true,
			results:                []cpmtypes.Result{permResult(cpmtypes.NonOK, testProblemReason, "boom")},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{Name: "problem_counter", Labels: map[string]string{"reason": testProblemReason}, Value: 1},
				{Name: "problem_gauge", Labels: map[string]string{"type": otherCondition, "reason": otherConditionOK}, Value: 0},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testProblemReason}, Value: 1},
			},
		},
		{
			name:                   "problem resolved does not increment the counter",
			enableMetricsReporting: true,
			results: []cpmtypes.Result{
				permResult(cpmtypes.NonOK, testProblemReason, "boom"),
				permResult(cpmtypes.OK, testProblemReason, "all good"),
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{Name: "problem_counter", Labels: map[string]string{"reason": testProblemReason}, Value: 1},
				{Name: "problem_gauge", Labels: map[string]string{"type": otherCondition, "reason": otherConditionOK}, Value: 0},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testProblemReason}, Value: 0},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testConditionOK}, Value: 0},
			},
		},
		{
			name:                   "unknown condition is not an active problem",
			enableMetricsReporting: true,
			results:                []cpmtypes.Result{permResult(cpmtypes.Unknown, testProblemReason, "cannot exec")},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{Name: "problem_gauge", Labels: map[string]string{"type": otherCondition, "reason": otherConditionOK}, Value: 0},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testConditionOK}, Value: 0},
			},
		},
		{
			name:                   "temporary problem increments the counter and refreshes every gauge",
			enableMetricsReporting: true,
			results: []cpmtypes.Result{{
				Rule:       &cpmtypes.CustomRule{Type: types.Temp, Reason: testTempReason},
				ExitStatus: cpmtypes.NonOK,
				Message:    "temporary problem detail",
			}},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{Name: "problem_counter", Labels: map[string]string{"reason": testTempReason}, Value: 1},
				{Name: "problem_gauge", Labels: map[string]string{"type": otherCondition, "reason": otherConditionOK}, Value: 0},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testConditionOK}, Value: 0},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			fakeCounter, fakeGauge := stubProblemMetrics(t)
			c := newTestMonitor(t, testOptions{
				defaultConditions:      defaultTestConditions(),
				enableMetricsReporting: test.enableMetricsReporting,
			})
			for _, result := range test.results {
				c.generateStatus(result)
			}
			gotMetrics := append(fakeCounter.ListMetrics(), fakeGauge.ListMetrics()...)
			assert.ElementsMatch(t, test.expectedMetrics, gotMetrics,
				"expected metrics: %+v, got: %+v", test.expectedMetrics, gotMetrics)
		})
	}
}

func TestGenerateStatusSurvivesMetricsFailure(t *testing.T) {
	original := problemmetrics.GlobalProblemMetricsManager
	t.Cleanup(func() { problemmetrics.GlobalProblemMetricsManager = original })
	problemmetrics.GlobalProblemMetricsManager = &problemmetrics.ProblemMetricsManager{}

	c := newTestMonitor(t, testOptions{
		defaultConditions:      defaultTestConditions(),
		enableMetricsReporting: true,
	})
	var got *types.Status
	require.NotPanics(t, func() {
		got = c.generateStatus(permResult(cpmtypes.NonOK, testProblemReason, "boom"))
	})
	// A metrics backend failure must not change what is reported.
	require.Len(t, got.Conditions, 2)
	assert.Equal(t, types.True, got.Conditions[1].Status)
	assert.Equal(t, testProblemReason, got.Conditions[1].Reason)
	assert.Equal(t, "boom", got.Conditions[1].Message)
	require.Len(t, got.Events, 1)
	assert.Equal(t, types.Warn, got.Events[0].Severity)
}

func TestGenerateStatusPreservesConfiguredDefaults(t *testing.T) {
	c := newTestMonitor(t, testOptions{defaultConditions: defaultTestConditions()})
	wantReason := c.config.DefaultConditions[1].Reason
	wantMessage := c.config.DefaultConditions[1].Message

	c.generateStatus(permResult(cpmtypes.NonOK, testProblemReason, "boom"))
	got := c.generateStatus(permResult(cpmtypes.OK, testProblemReason, "all good"))

	require.Len(t, got.Conditions, 2)
	assert.Equal(t, types.False, got.Conditions[1].Status)
	assert.Equal(t, wantReason, got.Conditions[1].Reason)
	assert.Equal(t, wantMessage, got.Conditions[1].Message)

	assert.Equal(t, wantReason, c.config.DefaultConditions[1].Reason)
	assert.Equal(t, wantMessage, c.config.DefaultConditions[1].Message)
	assert.Empty(t, c.config.DefaultConditions[1].Status)
}

func TestMonitorLoopReportsPluginResults(t *testing.T) {
	// A one-day interval makes sure the plugin runs only once.
	interval := 24 * time.Hour
	intervalString := interval.String()
	rule := &cpmtypes.CustomRule{
		Type:                 types.Perm,
		Condition:            testCondition,
		Reason:               testProblemReason,
		Path:                 testPluginScript("non-ok"),
		InvokeIntervalString: &intervalString,
	}
	c := newTestMonitor(t, testOptions{
		defaultConditions: defaultTestConditions(),
		rules:             []*cpmtypes.CustomRule{rule},
		skipInitialStatus: true,
	})
	statusChan, err := c.Start()
	require.NoError(t, err)

	status := receiveStatus(t, statusChan)
	require.Len(t, status.Conditions, 2)
	assert.Equal(t, types.True, status.Conditions[1].Status)
	assert.Equal(t, testProblemReason, status.Conditions[1].Reason)
	require.Len(t, status.Events, 1)
	assert.Equal(t, types.Warn, status.Events[0].Severity)

	requireStop(t, c)

	// Stop must also shut down the plugin it owns.
	select {
	case _, open := <-c.plugin.GetResultChan():
		assert.False(t, open, "Stop did not close the plugin result channel")
	case <-time.After(testWait):
		t.Fatal("Stop did not close the plugin result channel")
	}
}

func TestMonitorLoopExitsWhenResultChannelCloses(t *testing.T) {
	c := newTestMonitor(t, testOptions{defaultConditions: defaultTestConditions()})
	// monitorLoop must initialize the conditions itself.
	c.conditions = nil
	startedAt := time.Now()
	// Run monitorLoop directly; Stop would deadlock on the closed-channel path.
	go c.plugin.Run()
	returned := make(chan struct{})
	go func() {
		c.monitorLoop()
		close(returned)
	}()

	status := receiveStatus(t, c.statusChan)
	assert.Equal(t, testSource, status.Source)
	assert.Empty(t, status.Events)
	require.Len(t, status.Conditions, 2)
	for _, cond := range status.Conditions {
		assert.Equal(t, types.False, cond.Status)
		assert.WithinRange(t, cond.Transition, startedAt, time.Now())
	}

	// Stopping the plugin closes its result channel.
	c.plugin.Stop()
	requireNoStatus(t, c.statusChan)
	select {
	case <-returned:
	case <-time.After(testWait):
		t.Fatal("monitorLoop did not return after the result channel closed")
	}
}

func TestInitializeProblemMetricsOrDie(t *testing.T) {
	testCases := []struct {
		name            string
		rules           []*cpmtypes.CustomRule
		expectedMetrics []metrics.Int64MetricRepresentation
	}{
		{
			name:            "no rules",
			rules:           nil,
			expectedMetrics: []metrics.Int64MetricRepresentation{},
		},
		{
			name:  "temporary rule gets a counter but no gauge",
			rules: []*cpmtypes.CustomRule{{Type: types.Temp, Reason: testTempReason}},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{Name: "problem_counter", Labels: map[string]string{"reason": testTempReason}, Value: 0},
			},
		},
		{
			name: "permanent rule gets a counter and a gauge",
			rules: []*cpmtypes.CustomRule{
				{Type: types.Perm, Condition: testCondition, Reason: testProblemReason},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{Name: "problem_counter", Labels: map[string]string{"reason": testProblemReason}, Value: 0},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testProblemReason}, Value: 0},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			fakeCounter, fakeGauge := stubProblemMetrics(t)
			initializeProblemMetricsOrDie(test.rules)
			gotMetrics := append(fakeCounter.ListMetrics(), fakeGauge.ListMetrics()...)
			assert.ElementsMatch(t, test.expectedMetrics, gotMetrics,
				"expected metrics: %+v, got: %+v", test.expectedMetrics, gotMetrics)
		})
	}
}

func TestNewCustomPluginMonitorOrDie(t *testing.T) {
	fakeCounter, fakeGauge := stubProblemMetrics(t)

	raw, err := json.Marshal(map[string]any{
		"plugin": "custom",
		"pluginConfig": map[string]any{
			"invoke_interval": "10s",
			"timeout":         "5s",
		},
		"source": testSource,
		"conditions": []map[string]any{
			{"type": testCondition, "reason": testConditionOK, "message": testConditionOKMsg},
		},
		"rules": []map[string]any{
			{
				"type":      "permanent",
				"condition": testCondition,
				"reason":    testProblemReason,
				"path":      testPluginScript("ok"),
			},
		},
	})
	require.NoError(t, err)
	configPath := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(configPath, raw, 0o600))

	monitor := NewCustomPluginMonitorOrDie(configPath)
	require.NotNil(t, monitor)
	c, ok := monitor.(*customPluginMonitor)
	require.True(t, ok)

	assert.Equal(t, configPath, c.configPath)
	assert.Equal(t, testSource, c.config.Source)
	require.Len(t, c.config.Rules, 1)
	assert.Equal(t, testProblemReason, c.config.Rules[0].Reason)
	// ApplyConfiguration must set these pointers before use.
	require.NotNil(t, c.config.PluginGlobalConfig.InvokeInterval)
	assert.Equal(t, 10*time.Second, *c.config.PluginGlobalConfig.InvokeInterval)
	require.NotNil(t, c.config.PluginGlobalConfig.Timeout)
	require.NotNil(t, c.config.PluginGlobalConfig.Concurrency)
	require.NotNil(t, c.config.PluginGlobalConfig.MaxOutputLength)
	require.NotNil(t, c.config.PluginGlobalConfig.SkipInitialStatus)
	require.NotNil(t, c.config.EnableMetricsReporting)
	assert.True(t, *c.config.EnableMetricsReporting)
	assert.NotNil(t, c.plugin)
	assert.NotNil(t, c.tomb)
	// The send in monitorLoop is the only writer and must not block.
	assert.NotZero(t, cap(c.statusChan))

	expectedMetrics := []metrics.Int64MetricRepresentation{
		{Name: "problem_counter", Labels: map[string]string{"reason": testProblemReason}, Value: 0},
		{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testProblemReason}, Value: 0},
	}
	gotMetrics := append(fakeCounter.ListMetrics(), fakeGauge.ListMetrics()...)
	assert.ElementsMatch(t, expectedMetrics, gotMetrics)
}
