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

package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s.io/node-problem-detector/pkg/types"
)

type fakeConditionsGetter struct {
	conditions []types.Condition
}

func (f *fakeConditionsGetter) GetConditions() []types.Condition {
	return f.conditions
}

// get issues a request against the handler and returns the status and body.
func get(t *testing.T, handler http.Handler, path string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// The diagnostic endpoints are controlled by --port alone, so they must be
// served whether or not the Kubernetes exporter supplied a conditions getter.
func TestHandlerEndpointsServedWithoutConditionsGetter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		getter ConditionsGetter
	}{
		{"with getter", &fakeConditionsGetter{}},
		{"without getter", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewHandler(tc.getter)

			if code, body := get(t, handler, "/healthz"); code != http.StatusOK || body != "ok" {
				t.Errorf("/healthz: wanted 200 %q, got %d %q", "ok", code, body)
			}
			for _, path := range []string{"/debug/pprof/", "/debug/pprof/cmdline", "/conditions"} {
				if code, _ := get(t, handler, path); code != http.StatusOK {
					t.Errorf("%s: wanted 200, got %d", path, code)
				}
			}
		})
	}
}

func TestHandlerConditions(t *testing.T) {
	want := []types.Condition{{
		Type:       "TestCondition",
		Status:     types.True,
		Transition: time.Date(2026, time.August, 26, 10, 0, 0, 0, time.UTC),
		Reason:     "TestReason",
		Message:    "test message",
	}}

	code, body := get(t, NewHandler(&fakeConditionsGetter{conditions: want}), "/conditions")
	if code != http.StatusOK {
		t.Fatalf("/conditions: wanted 200, got %d", code)
	}
	var got []types.Condition
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("/conditions returned undecodable body %q: %v", body, err)
	}
	if len(got) != 1 || got[0].Type != want[0].Type || got[0].Status != want[0].Status ||
		got[0].Reason != want[0].Reason || got[0].Message != want[0].Message {
		t.Errorf("/conditions: wanted %+v, got %+v", want, got)
	}
}

// Without a getter there is no condition to report. The response must match an
// enabled exporter that has not recorded any condition yet, so that a client
// cannot tell the two apart by parsing the body.
func TestHandlerConditionsWithoutGetterMatchesEmptyExporter(t *testing.T) {
	_, withoutGetter := get(t, NewHandler(nil), "/conditions")
	_, emptyGetter := get(t, NewHandler(&fakeConditionsGetter{}), "/conditions")

	if withoutGetter != emptyGetter {
		t.Errorf("/conditions without a getter returned %q, but an exporter with no conditions returned %q",
			withoutGetter, emptyGetter)
	}
}
