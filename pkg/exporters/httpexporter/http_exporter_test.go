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

package httpexporter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/types"
)

func newTestExporter() *httpExporter {
	return &httpExporter{conditions: make(map[string]types.Condition)}
}

func newTestCondition(conditionType, reason string, status types.ConditionStatus) types.Condition {
	return types.Condition{
		Type:       conditionType,
		Status:     status,
		Transition: time.Now(),
		Reason:     reason,
		Message:    "test message",
	}
}

func TestNewExporterOrDieDisabled(t *testing.T) {
	testCases := []struct {
		name string
		port int
	}{
		{
			name: "Zero port disables the exporter",
			port: 0,
		},
		{
			name: "Negative port disables the exporter",
			port: -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			npdo := &options.NodeProblemDetectorOptions{
				ServerPort:    test.port,
				ServerAddress: "127.0.0.1",
			}
			assert.Nil(t, NewExporterOrDie(npdo))
		})
	}
}

func TestExportProblems(t *testing.T) {
	// The exporter is shared across cases so that each case builds on the
	// state left behind by the previous one.
	he := newTestExporter()

	testCases := []struct {
		name string
		// status is exported at the start of the case.
		status *types.Status
		// wantReasons is the full expected condition store afterwards, keyed by
		// condition type.
		wantReasons map[string]string
	}{
		{
			name: "First condition is stored",
			status: &types.Status{
				Source:     "test",
				Conditions: []types.Condition{newTestCondition("TypeA", "ReasonA", types.True)},
			},
			wantReasons: map[string]string{"TypeA": "ReasonA"},
		},
		{
			name: "Condition of a new type is stored alongside the existing one",
			status: &types.Status{
				Source:     "test",
				Conditions: []types.Condition{newTestCondition("TypeB", "ReasonB", types.True)},
			},
			wantReasons: map[string]string{"TypeA": "ReasonA", "TypeB": "ReasonB"},
		},
		{
			name: "Condition of a known type overrides instead of appending",
			status: &types.Status{
				Source:     "test",
				Conditions: []types.Condition{newTestCondition("TypeA", "ReasonAUpdated", types.False)},
			},
			wantReasons: map[string]string{"TypeA": "ReasonAUpdated", "TypeB": "ReasonB"},
		},
		{
			name: "Multiple conditions in one status are all stored",
			status: &types.Status{
				Source: "test",
				Conditions: []types.Condition{
					newTestCondition("TypeC", "ReasonC", types.True),
					newTestCondition("TypeD", "ReasonD", types.True),
				},
			},
			wantReasons: map[string]string{
				"TypeA": "ReasonAUpdated", "TypeB": "ReasonB",
				"TypeC": "ReasonC", "TypeD": "ReasonD",
			},
		},
		{
			name: "Events do not affect the condition store",
			status: &types.Status{
				Source: "test",
				Events: []types.Event{{
					Severity:  types.Warn,
					Timestamp: time.Now(),
					Reason:    "TestEvent",
					Message:   "test event message",
				}},
			},
			wantReasons: map[string]string{
				"TypeA": "ReasonAUpdated", "TypeB": "ReasonB",
				"TypeC": "ReasonC", "TypeD": "ReasonD",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			he.ExportProblems(test.status)

			gotReasons := map[string]string{}
			for _, c := range he.getConditions() {
				gotReasons[c.Type] = c.Reason
			}
			assert.Equal(t, test.wantReasons, gotReasons)
		})
	}
}

func TestHandlers(t *testing.T) {
	testCases := []struct {
		name string
		// seed is exported before the request is served.
		seed            []types.Condition
		path            string
		wantStatus      int
		wantContentType string
		// wantBody is compared exactly when set.
		wantBody string
	}{
		{
			name:       "healthz always reports ok",
			path:       "/healthz",
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name:            "conditions serves an empty array when nothing was exported",
			path:            "/conditions",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantBody:        "[]",
		},
		{
			name:       "pprof index is registered",
			path:       "/debug/pprof/",
			wantStatus: http.StatusOK,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			he := newTestExporter()
			if len(test.seed) != 0 {
				he.ExportProblems(&types.Status{Source: "test", Conditions: test.seed})
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, test.path, nil)
			he.buildMux().ServeHTTP(w, req)

			assert.Equal(t, test.wantStatus, w.Code)
			if test.wantContentType != "" {
				assert.Equal(t, test.wantContentType, w.Header().Get("Content-type"))
			}
			if test.wantBody != "" {
				assert.Equal(t, test.wantBody, w.Body.String())
			}
		})
	}
}

func TestConditionsHandlerServesExportedCondition(t *testing.T) {
	he := newTestExporter()
	he.ExportProblems(&types.Status{
		Source:     "test",
		Conditions: []types.Condition{newTestCondition("KernelDeadlock", "AUFSUmountHung", types.True)},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/conditions", nil)
	he.buildMux().ServeHTTP(w, req)

	var conditions []types.Condition
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &conditions))
	assert.Len(t, conditions, 1)
	assert.Equal(t, "KernelDeadlock", conditions[0].Type)
	assert.Equal(t, "AUFSUmountHung", conditions[0].Reason)
	assert.Equal(t, types.True, conditions[0].Status)
}

func TestConcurrentAccess(t *testing.T) {
	he := newTestExporter()
	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			for range iterations {
				he.ExportProblems(&types.Status{
					Source:     "test",
					Conditions: []types.Condition{newTestCondition(fmt.Sprintf("Type%d", i), "Reason", types.True)},
				})
			}
		}(i)
		go func() {
			defer wg.Done()
			for range iterations {
				he.getConditions()
			}
		}()
	}
	wg.Wait()

	assert.Len(t, he.getConditions(), goroutines)
}
