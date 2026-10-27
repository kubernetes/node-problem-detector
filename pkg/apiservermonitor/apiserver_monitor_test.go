/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package apiservermonitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemmetrics"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/metrics"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

const (
	testSource         = "apiserver-monitor-test"
	testConfigPath     = "test-config-path"
	testCondition      = "APIServerUnreachable"
	testConditionOK    = "APIServerIsReachable"
	testConditionOKMsg = "apiserver is reachable"
	testWait           = 3 * time.Second
)

var testEpoch = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

func TestRegistration(t *testing.T) {
	assert.NotPanics(t,
		func() { problemdaemon.GetProblemDaemonHandlerOrDie(APIServerMonitorName) },
		"APIServer monitor failed to register itself as a problem daemon.")
}

type testOptions struct {
	serverURL              string
	invokeInterval         time.Duration
	timeout                time.Duration
	defaultConditions      []types.Condition
	enableMetricsReporting bool
	skipInitialStatus      bool
}

func defaultTestConditions() []types.Condition {
	return []types.Condition{
		{Type: testCondition, Reason: testConditionOK, Message: testConditionOKMsg},
	}
}

func newTestMonitor(t *testing.T, o testOptions) *apiServerMonitor {
	t.Helper()
	if o.invokeInterval == 0 {
		o.invokeInterval = 10 * time.Second
	}
	if o.timeout == 0 {
		o.timeout = 5 * time.Second
	}
	if len(o.defaultConditions) == 0 {
		o.defaultConditions = defaultTestConditions()
	}

	cfg := MonitorConfig{
		InvokeIntervalString:   o.invokeInterval.String(),
		TimeoutString:          o.timeout.String(),
		Source:                 testSource,
		DefaultConditions:      o.defaultConditions,
		EnableMetricsReporting: &o.enableMetricsReporting,
		SkipInitialStatus:      &o.skipInitialStatus,
	}
	require.NoError(t, cfg.ApplyConfiguration())
	require.NoError(t, cfg.Validate())

	serverURL := o.serverURL
	if serverURL == "" {
		serverURL = "http://127.0.0.1:8080"
	}
	kubeConfig, err := util.NewKubeClientConfig(fmt.Sprintf("%s?inClusterConfig=false", serverURL), 500, 500)
	require.NoError(t, err)
	kubeConfig.Timeout = cfg.Timeout
	if kubeConfig.Proxy == nil {
		kubeConfig.Proxy = http.ProxyFromEnvironment
	}

	a := &apiServerMonitor{
		configPath: testConfigPath,
		config:     cfg,
		kubeConfig: kubeConfig,
		statusChan: make(chan *types.Status, 1000),
		tomb:       tomb.NewTomb(),
	}
	a.initializeConditions()
	for i := range a.conditions {
		a.conditions[i].Transition = testEpoch
	}
	return a
}

func stubProblemMetrics(t *testing.T) (*metrics.FakeInt64Metric, *metrics.FakeInt64Metric) {
	t.Helper()
	original := problemmetrics.GlobalProblemMetricsManager
	t.Cleanup(func() { problemmetrics.GlobalProblemMetricsManager = original })
	pmm, counter, gauge := problemmetrics.NewProblemMetricsManagerStub()
	problemmetrics.GlobalProblemMetricsManager = pmm
	return counter, gauge
}

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

func requireStop(t *testing.T, a *apiServerMonitor) {
	t.Helper()
	stopped := make(chan struct{})
	go func() {
		a.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(testWait):
		t.Fatal("Stop did not return")
	}
}

func TestCheckAPIServer(t *testing.T) {
	var requestedPath atomic.Value
	var statusCode atomic.Int32
	var newConnCount atomic.Int32
	statusCode.Store(http.StatusOK)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath.Store(r.URL.Path)
		code := int(statusCode.Load())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		if code == http.StatusOK {
			_, _ = w.Write([]byte(`{"major":"1","minor":"30"}`))
		} else {
			_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"internal error","code":500}`))
		}
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			newConnCount.Add(1)
		}
	}
	server.StartTLS()
	t.Cleanup(server.Close)

	a := newTestMonitor(t, testOptions{serverURL: server.URL})
	a.kubeConfig.TLSClientConfig.Insecure = true

	err := a.checkAPIServer(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "/version", requestedPath.Load())
	assert.Equal(t, int32(1), newConnCount.Load(), "first check should establish a new TCP connection")

	// Second check should establish a fresh TCP connection rather than reusing a cached transport's connection.
	err = a.checkAPIServer(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(2), newConnCount.Load(), "second check should establish a new TCP connection")

	statusCode.Store(http.StatusInternalServerError)
	err = a.checkAPIServer(context.Background())
	require.Error(t, err)
	assert.Equal(t, int32(3), newConnCount.Load(), "third check should establish a new TCP connection")

	server.Close()
	err = a.checkAPIServer(context.Background())
	require.Error(t, err)
}

func TestGenerateStatusForConditions(t *testing.T) {
	type step struct {
		checkErr    error
		wantStatus  types.ConditionStatus
		wantReason  string
		wantMessage string
		wantEvent   bool
	}

	errBoom := errors.New("Get \"http://127.0.0.1:8080/version\": dial tcp 127.0.0.1:8080: connect: connection refused")

	testCases := []struct {
		name  string
		steps []step
	}{
		{
			name: "stays false when healthy, no event",
			steps: []step{
				{
					checkErr:    nil,
					wantStatus:  types.False,
					wantReason:  testConditionOK,
					wantMessage: testConditionOKMsg,
					wantEvent:   false,
				},
				{
					checkErr:    nil,
					wantStatus:  types.False,
					wantReason:  testConditionOK,
					wantMessage: testConditionOKMsg,
					wantEvent:   false,
				},
			},
		},
		{
			name: "false to true then back to false restores default",
			steps: []step{
				{
					checkErr:    errBoom,
					wantStatus:  types.True,
					wantReason:  testCondition,
					wantMessage: errBoom.Error(),
					wantEvent:   true,
				},
				{
					checkErr:    errBoom,
					wantStatus:  types.True,
					wantReason:  testCondition,
					wantMessage: errBoom.Error(),
					wantEvent:   false,
				},
				{
					checkErr:    nil,
					wantStatus:  types.False,
					wantReason:  testConditionOK,
					wantMessage: testConditionOKMsg,
					wantEvent:   true,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			a := newTestMonitor(t, testOptions{})
			for i, s := range test.steps {
				got := a.generateStatus(s.checkErr)

				require.Len(t, got.Conditions, 1, "step %d", i)
				assert.Equal(t, testSource, got.Source, "step %d", i)

				cond := got.Conditions[0]
				assert.Equal(t, testCondition, cond.Type, "step %d", i)
				assert.Equal(t, s.wantStatus, cond.Status, "step %d", i)
				assert.Equal(t, s.wantReason, cond.Reason, "step %d", i)
				assert.Equal(t, s.wantMessage, cond.Message, "step %d", i)

				if !s.wantEvent {
					assert.Empty(t, got.Events, "step %d", i)
					continue
				}
				require.Len(t, got.Events, 1, "step %d", i)
				event := got.Events[0]
				wantSeverity := types.Info
				if s.wantStatus == types.True {
					wantSeverity = types.Warn
				}
				assert.Equal(t, wantSeverity, event.Severity, "step %d", i)
				want := util.GenerateConditionChangeEvent(
					testCondition, s.wantStatus, s.wantReason, s.wantMessage, event.Timestamp)
				assert.Equal(t, want, event, "step %d", i)
				assert.True(t, cond.Transition.Equal(event.Timestamp), "step %d", i)
				assert.False(t, cond.Transition.Equal(testEpoch), "step %d", i)
			}
		})
	}
}

func TestGenerateStatusMetrics(t *testing.T) {
	errBoom := errors.New("connection refused")

	testCases := []struct {
		name                   string
		enableMetricsReporting bool
		checkErrs              []error
		expectedMetrics        []metrics.Int64MetricRepresentation
	}{
		{
			name:                   "reporting disabled records nothing",
			enableMetricsReporting: false,
			checkErrs:              []error{errBoom},
			expectedMetrics:        []metrics.Int64MetricRepresentation{},
		},
		{
			name:                   "problem detected increments counter and raises gauge",
			enableMetricsReporting: true,
			checkErrs:              []error{errBoom},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{Name: "problem_counter", Labels: map[string]string{"reason": testCondition}, Value: 1},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testCondition}, Value: 1},
			},
		},
		{
			name:                   "problem resolved lowers gauge without incrementing counter",
			enableMetricsReporting: true,
			checkErrs:              []error{errBoom, nil},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{Name: "problem_counter", Labels: map[string]string{"reason": testCondition}, Value: 1},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testCondition}, Value: 0},
				{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testConditionOK}, Value: 0},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			fakeCounter, fakeGauge := stubProblemMetrics(t)
			a := newTestMonitor(t, testOptions{
				enableMetricsReporting: test.enableMetricsReporting,
			})
			for _, checkErr := range test.checkErrs {
				a.generateStatus(checkErr)
			}
			gotMetrics := append(fakeCounter.ListMetrics(), fakeGauge.ListMetrics()...)
			assert.ElementsMatch(t, test.expectedMetrics, gotMetrics)
		})
	}
}

func TestMonitorLoop(t *testing.T) {
	var healthy atomic.Bool
	healthy.Store(false)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if healthy.Load() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"major":"1","minor":"30"}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	a := newTestMonitor(t, testOptions{
		serverURL:         server.URL,
		invokeInterval:    50 * time.Millisecond,
		timeout:           25 * time.Millisecond,
		skipInitialStatus: false,
	})

	statusChan, err := a.Start()
	require.NoError(t, err)

	// 1. Initial status is False with no events.
	initial := receiveStatus(t, statusChan)
	require.Len(t, initial.Conditions, 1)
	assert.Equal(t, types.False, initial.Conditions[0].Status)
	assert.Empty(t, initial.Events)

	// 2. First check fails -> condition becomes True with a Warn event.
	firstCheck := receiveStatus(t, statusChan)
	require.Len(t, firstCheck.Conditions, 1)
	assert.Equal(t, types.True, firstCheck.Conditions[0].Status)
	assert.Equal(t, testCondition, firstCheck.Conditions[0].Reason)
	require.Len(t, firstCheck.Events, 1)
	assert.Equal(t, types.Warn, firstCheck.Events[0].Severity)

	// 3. Server becomes healthy -> condition returns to False with an Info event.
	healthy.Store(true)
	var recovered *types.Status
	for i := 0; i < 20; i++ {
		st := receiveStatus(t, statusChan)
		if len(st.Conditions) == 1 && st.Conditions[0].Status == types.False {
			recovered = st
			break
		}
	}
	require.NotNil(t, recovered, "expected monitor to recover once server returned 200 OK")
	assert.Equal(t, testConditionOK, recovered.Conditions[0].Reason)
	require.Len(t, recovered.Events, 1)
	assert.Equal(t, types.Info, recovered.Events[0].Severity)

	requireStop(t, a)
}

func TestNewAPIServerMonitorOrDie(t *testing.T) {
	fakeCounter, fakeGauge := stubProblemMetrics(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	originalCfg := util.GlobalKubeClientConfig
	t.Cleanup(func() { util.GlobalKubeClientConfig = originalCfg })
	globalCfg, err := util.NewKubeClientConfig(fmt.Sprintf("%s?inClusterConfig=false", server.URL), 123, 456)
	require.NoError(t, err)
	util.GlobalKubeClientConfig = globalCfg

	raw, err := json.Marshal(map[string]any{
		"invokeInterval":   "15s",
		"timeout":          "4s",
		"source":           testSource,
		"metricsReporting": true,
		"conditions": []map[string]any{
			{"type": testCondition, "reason": testConditionOK, "message": testConditionOKMsg},
		},
	})
	require.NoError(t, err)
	configPath := filepath.Join(t.TempDir(), "apiserver-monitor.json")
	require.NoError(t, os.WriteFile(configPath, raw, 0o600))

	monitor := NewAPIServerMonitorOrDie(configPath)
	require.NotNil(t, monitor)
	a, ok := monitor.(*apiServerMonitor)
	require.True(t, ok)

	assert.Equal(t, configPath, a.configPath)
	assert.Equal(t, testSource, a.config.Source)
	assert.Equal(t, 15*time.Second, a.config.InvokeInterval)
	assert.Equal(t, 4*time.Second, a.config.Timeout)
	require.NotNil(t, a.kubeConfig)
	assert.Equal(t, server.URL, a.kubeConfig.Host)
	assert.Equal(t, float32(123), a.kubeConfig.QPS)
	assert.Equal(t, 456, a.kubeConfig.Burst)
	assert.Equal(t, 4*time.Second, a.kubeConfig.Timeout)
	assert.NotNil(t, a.kubeConfig.Proxy)
	assert.Equal(t, time.Duration(0), util.GlobalKubeClientConfig.Timeout)
	assert.Nil(t, util.GlobalKubeClientConfig.Proxy)
	assert.NotZero(t, cap(a.statusChan))

	expectedMetrics := []metrics.Int64MetricRepresentation{
		{Name: "problem_counter", Labels: map[string]string{"reason": testCondition}, Value: 0},
		{Name: "problem_gauge", Labels: map[string]string{"type": testCondition, "reason": testCondition}, Value: 0},
	}
	gotMetrics := append(fakeCounter.ListMetrics(), fakeGauge.ListMetrics()...)
	assert.ElementsMatch(t, expectedMetrics, gotMetrics)
}
