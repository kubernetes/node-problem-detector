/*
Copyright 2025 The Kubernetes Authors All rights reserved.

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

package metrics

import (
	"testing"
)

func TestGaugeSetValueSemantics(t *testing.T) {
	// Create a gauge metric
	gauge, err := NewInt64Metric("test_gauge", "test_gauge", "Test gauge metric", "1", LastValue, []string{"reason", "type"})
	if err != nil {
		t.Fatalf("Failed to create gauge metric: %v", err)
	}

	// Set initial value to 0 (initialization)
	labels1 := map[string]string{"reason": "TestReason", "type": "TestType"}
	err = gauge.Record(labels1, 0)
	if err != nil {
		t.Fatalf("Failed to record initial value: %v", err)
	}

	// Set value to 1 (problem detected)
	err = gauge.Record(labels1, 1)
	if err != nil {
		t.Fatalf("Failed to record updated value: %v", err)
	}

	// Set value back to 0 (problem resolved)
	err = gauge.Record(labels1, 0)
	if err != nil {
		t.Fatalf("Failed to record resolved value: %v", err)
	}

	// Test with different labels
	labels2 := map[string]string{"reason": "AnotherReason", "type": "TestType"}
	err = gauge.Record(labels2, 0)
	if err != nil {
		t.Fatalf("Failed to record value for different labels: %v", err)
	}
}

func TestCounterAddSemantics(t *testing.T) {
	// Create a counter metric
	counter, err := NewInt64Metric("test_counter", "test_counter", "Test counter metric", "1", Sum, []string{"reason"})
	if err != nil {
		t.Fatalf("Failed to create counter metric: %v", err)
	}

	// Initialize counter to 0
	labels := map[string]string{"reason": "TestReason"}
	err = counter.Record(labels, 0)
	if err != nil {
		t.Fatalf("Failed to record initial value: %v", err)
	}

	// Increment counter
	err = counter.Record(labels, 1)
	if err != nil {
		t.Fatalf("Failed to increment counter: %v", err)
	}
}
