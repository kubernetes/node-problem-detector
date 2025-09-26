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
	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

type prometheusExporter struct{}

// NewExporterOrDie creates an exporter to export metrics to Prometheus, panics if error occurs.
func NewExporterOrDie(npdo *options.NodeProblemDetectorOptions) types.Exporter {
	if npdo.PrometheusServerPort <= 0 {
		return nil
	}

	// Create Prometheus exporter with options to prevent automatic suffixing
	promExporter, err := prometheus.New(
		prometheus.WithoutCounterSuffixes(), // Don't add _total suffix to counters
		prometheus.WithoutUnits(),           // Don't add unit-based suffixes like _ratio
		prometheus.WithoutScopeInfo(),       // Don't add otel_scope_* labels
	)
	if err != nil {
		klog.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	// register with the global meter provider
	otelutil.AddMetricReader(promExporter)

	addr := net.JoinHostPort(npdo.PrometheusServerAddress, strconv.Itoa(npdo.PrometheusServerPort))
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(addr, mux); err != nil {
			klog.Fatalf("Failed to start Prometheus scrape endpoint: %v", err)
		}
	}()

	klog.Infof("Prometheus exporter started on %s", addr)
	return &prometheusExporter{}
}

// ExportProblems does nothing.
// Prometheus exporter only exports metrics.
func (pe *prometheusExporter) ExportProblems(status *types.Status) {
}
