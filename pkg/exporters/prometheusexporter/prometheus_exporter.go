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
	"net"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type prometheusExporter struct{}

// NewExporterOrDie creates an exporter to export metrics to Prometheus, panics if error occurs.
func NewExporterOrDie(npdo *options.NodeProblemDetectorOptions) types.Exporter {
	if npdo.PrometheusServerPort <= 0 {
		return nil
	}

	addr := net.JoinHostPort(npdo.PrometheusServerAddress, strconv.Itoa(npdo.PrometheusServerPort))
	// Use options to prevent OpenTelemetry from modifying metric names:
	// - WithoutUnits: prevents adding unit suffixes like "_ratio" to metric names
	// - WithoutCounterSuffixes: prevents adding "_total" suffix to counters
	// - WithoutScopeInfo: prevents adding otel_scope_* labels to metrics
	// This ensures backward compatibility with existing metric names.
	exporter, err := prometheus.New(
		prometheus.WithoutUnits(),
		prometheus.WithoutCounterSuffixes(),
		prometheus.WithoutScopeInfo(),
	)
	if err != nil {
		klog.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	// Register the exporter as a reader for metrics collection
	metrics.AddReader(exporter)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(addr, mux); err != nil {
			klog.Fatalf("Failed to start Prometheus scrape endpoint: %v", err)
		}
	}()

	return &prometheusExporter{}
}

// ExportProblems does nothing.
// Prometheus exporter only exports metrics.
func (pe *prometheusExporter) ExportProblems(status *types.Status) {
}
