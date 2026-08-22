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

package metrics

import "testing"

func TestMetricMappingKeepsPairedValues(t *testing.T) {
	mapping := MetricMapping{
		viewNameToMapping: make(map[string]metricMappingEntry),
	}

	if err := mapping.AddNormalizedMapping("custom", "custom_metric", "custom/metric"); err != nil {
		t.Fatalf("AddNormalizedMapping returned error: %v", err)
	}
	if err := mapping.AddNormalizedMapping("custom", "custom_metric", "custom/metric"); err != nil {
		t.Fatalf("idempotent AddNormalizedMapping returned error: %v", err)
	}

	if got, ok := mapping.ViewNameToMetricID("custom_metric"); !ok || got != "custom" {
		t.Fatalf("metric ID = %q, %t; want %q, true", got, ok, "custom")
	}
	if got, ok := mapping.ViewNameToOriginalName("custom_metric"); !ok || got != "custom/metric" {
		t.Fatalf("original name = %q, %t; want %q, true", got, ok, "custom/metric")
	}

	if err := mapping.AddNormalizedMapping("other", "custom_metric", "other/metric"); err == nil {
		t.Fatal("expected conflicting normalized mapping to return an error")
	}
}

func TestMetricMappingUsesViewNameAsOriginalName(t *testing.T) {
	mapping := MetricMapping{
		viewNameToMapping: make(map[string]metricMappingEntry),
	}

	mapping.AddMapping("host/uptime", "host/uptime")

	if got, ok := mapping.ViewNameToMetricID("host/uptime"); !ok || got != "host/uptime" {
		t.Fatalf("metric ID = %q, %t; want %q, true", got, ok, "host/uptime")
	}
	if got, ok := mapping.ViewNameToOriginalName("host/uptime"); !ok || got != "host/uptime" {
		t.Fatalf("original name = %q, %t; want %q, true", got, ok, "host/uptime")
	}
}
