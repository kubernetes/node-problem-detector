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
package metrics

import (
	"os"
	"testing"
)

// TestPrometheusMetricsParsingAndMatching verifies the behavior of ParsePrometheusMetrics() and GetFloat64Metric().
func TestPrometheusMetricsParsingAndMatching(t *testing.T) {
	testCases := []struct {
		name                string
		metricsTextPath     string
		expectedMetrics     []Float64MetricRepresentation
		notExpectedMetrics  []Float64MetricRepresentation
		strictLabelMatching bool
	}{
		{
			name:            "Relaxed label matching",
			metricsTextPath: "testdata/sample_metrics.txt",
			expectedMetrics: []Float64MetricRepresentation{
				// Metric with no label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{},
				},
				// Metric with partial label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"kernel_version": "4.14.127+"},
				},
				{
					Name:   "disk_avg_queue_len",
					Labels: map[string]string{"device": "sda1"},
				},
				{
					Name:   "disk_avg_queue_len",
					Labels: map[string]string{"device": "sda8"},
				},
			},
			notExpectedMetrics: []Float64MetricRepresentation{
				// Metric with non-existent label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"non-existent-version": "0.0.1"},
				},
				// Metric with incorrect label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"kernel_version": "mismatched-version"},
				},
				// Non-existent metric.
				{
					Name:   "host_downtime",
					Labels: map[string]string{},
				},
			},
			strictLabelMatching: false,
		},
		{
			name:            "Strict label matching",
			metricsTextPath: "testdata/sample_metrics.txt",
			expectedMetrics: []Float64MetricRepresentation{
				{
					Name:   "host_uptime",
					Labels: map[string]string{"kernel_version": "4.14.127+", "os_version": "cos 73-11647.217.0"},
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "DockerHung"},
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "OOMKilling"},
				},
			},
			notExpectedMetrics: []Float64MetricRepresentation{
				// Metric with incomplete label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"kernel_version": "4.14.127+"},
				},
				// Metric with missing label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{},
				},
				// Metric with non-existent label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"non-existent-version": "0.0.1"},
				},
				// Metric with incorrect label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"kernel_version": "mismatched-version"},
				},
				// Non-existent metric.
				{
					Name:   "host_downtime",
					Labels: map[string]string{},
				},
			},
			strictLabelMatching: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			b, err := os.ReadFile(test.metricsTextPath)
			if err != nil {
				t.Errorf("Unexpected error reading file %s: %v", test.metricsTextPath, err)
			}
			metricsText := string(b)

			metrics, err := ParsePrometheusMetrics(metricsText)
			if err != nil {
				t.Errorf("Unexpected error parsing NPD metrics: %v\nMetrics text: %s\n", err, metricsText)
			}

			for _, expectedMetric := range test.expectedMetrics {
				_, err = GetFloat64Metric(metrics, expectedMetric.Name, expectedMetric.Labels, test.strictLabelMatching)
				if err != nil {
					t.Errorf("Failed to find metric %v in these metrics %v.\nMetrics text: %s\n",
						expectedMetric, metrics, metricsText)
				}
			}

			for _, notExpectedMetric := range test.notExpectedMetrics {
				_, err = GetFloat64Metric(metrics, notExpectedMetric.Name, notExpectedMetric.Labels, test.strictLabelMatching)
				if err == nil {
					t.Errorf("Unexpected metric %v found in these metrics %v.\nMetrics text: %s\n",
						notExpectedMetric, metrics, metricsText)
				}
			}
		})
	}
}

func TestParsePrometheusMetricsSkipsUnsupportedTypes(t *testing.T) {
	testCases := []struct {
		name        string
		metricsText string
	}{
		{
			name: "Explicit untyped",
			metricsText: `# TYPE a_untyped UNTYPED
a_untyped 3
# TYPE b_gauge GAUGE
b_gauge 7
`,
		},
		{
			name: "Implicit untyped",
			metricsText: `kpid_total 3
# TYPE b_gauge GAUGE
b_gauge 7
`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			metrics, err := ParsePrometheusMetrics(test.metricsText)
			if err != nil {
				t.Fatalf("Unexpected error parsing metrics: %v", err)
			}
			if len(metrics) != 1 || metrics[0].Name != "b_gauge" || metrics[0].Value != 7 {
				t.Fatalf("Expected only b_gauge=7, got %#v", metrics)
			}
		})
	}
}

func TestParsePrometheusMetricsPreservesSampleSums(t *testing.T) {
	testCases := []struct {
		name        string
		metricsText string
		metricName  string
		wantValue   float64
	}{
		{
			name: "Histogram",
			metricsText: `# TYPE a_hist HISTOGRAM
a_hist_bucket{le="1"} 1
a_hist_bucket{le="+Inf"} 1
a_hist_sum 1
a_hist_count 1
`,
			metricName: "a_hist",
			wantValue:  1,
		},
		{
			name: "Summary",
			metricsText: `# TYPE a_summary SUMMARY
a_summary{quantile="0.5"} 1
a_summary_sum 2
a_summary_count 1
`,
			metricName: "a_summary",
			wantValue:  2,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			metrics, err := ParsePrometheusMetrics(test.metricsText)
			if err != nil {
				t.Fatalf("Unexpected error parsing metrics: %v", err)
			}
			if len(metrics) != 1 || metrics[0].Name != test.metricName || metrics[0].Value != test.wantValue {
				t.Fatalf("Expected only %s=%v, got %#v", test.metricName, test.wantValue, metrics)
			}
		})
	}
}
