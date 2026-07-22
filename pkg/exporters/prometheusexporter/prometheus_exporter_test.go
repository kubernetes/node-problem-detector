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

package prometheusexporter

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"k8s.io/node-problem-detector/pkg/util/metrics"
	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

// TestScrapeExcludesDefaultCollectors verifies that the Prometheus scrape
// endpoint exports NPD's own metrics but does NOT include the Go runtime and
// process collectors that the global prometheus.DefaultRegisterer registers by
// default.
func TestScrapeExcludesDefaultCollectors(t *testing.T) {
	otelutil.ResetForTesting()
	defer otelutil.ResetForTesting()

	// Build the exporter and handler exactly like production does.
	promExporter, handler, err := newExporterAndHandler()
	if err != nil {
		t.Fatalf("Failed to create Prometheus exporter: %v", err)
	}
	otelutil.AddMetricReader(promExporter)
	otelutil.InitializeMeterProvider()

	// Record a metric through the standard NPD metrics + otel path.
	metric, err := metrics.NewInt64Metric(
		metrics.HostUptimeID,
		"host/uptime",
		"system uptime",
		"s",
		metrics.LastValue,
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create metric: %v", err)
	}
	if err := metric.Record(nil, 1); err != nil {
		t.Fatalf("Failed to record metric: %v", err)
	}

	// Scrape the handler.
	server := httptest.NewServer(handler)
	defer server.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create scrape request: %v", err)
	}
	req.Header.Set("Accept", "text/plain; version=1.0.0; escaping=allow-utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to scrape metrics endpoint: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close scrape response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read scrape response: %v", err)
	}
	output := string(body)

	// NPD metric names must retain the legacy underscore escaping even when
	// Prometheus 3 requests UTF-8 names.
	if !strings.Contains(output, "host_uptime") {
		t.Errorf("Expected scrape output to contain recorded metric %q, got:\n%s", "host_uptime", output)
	}
	if strings.Contains(output, "host/uptime") {
		t.Errorf("Expected scrape output to escape metric name %q, got:\n%s", "host/uptime", output)
	}

	// The default Go runtime / process collectors and the target_info metric
	// (which would expose the shared OTel resource attributes) must NOT be
	// present.
	for _, unwanted := range []string{"go_goroutines", "process_cpu_seconds_total", "target_info"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("Expected scrape output to NOT contain metric %q, got:\n%s", unwanted, output)
		}
	}
}
