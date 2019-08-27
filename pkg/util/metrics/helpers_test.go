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
	"io/ioutil"
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
				// Metric with non-existant label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"non-existant-version": "0.0.1"},
				},
				// Metric with incorrect label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"kernel_version": "mismatched-version"},
				},
				// Non-exsistant metric.
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
				// Metric with non-existant label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"non-existant-version": "0.0.1"},
				},
				// Metric with incorrect label.
				{
					Name:   "host_uptime",
					Labels: map[string]string{"kernel_version": "mismatched-version"},
				},
				// Non-exsistant metric.
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
			b, err := ioutil.ReadFile(test.metricsTextPath)
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
