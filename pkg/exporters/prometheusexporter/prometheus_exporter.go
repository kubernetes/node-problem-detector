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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/otlptranslator"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/types"
	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

type prometheusExporter struct{}

// newExporterAndHandler creates the OTel Prometheus exporter backed by a
// dedicated registry and returns an HTTP handler that scrapes only that
// registry. Using a dedicated registry (instead of the global
// prometheus.DefaultRegisterer) keeps the default Go runtime and process
// collectors out of NPD's scrape output.
func newExporterAndHandler() (*otelprometheus.Exporter, http.Handler, error) {
	reg := prometheus.NewRegistry()

	// Create Prometheus exporter with options to prevent automatic suffixing
	promExporter, err := otelprometheus.New(
		otelprometheus.WithRegisterer(reg),
		otelprometheus.WithTranslationStrategy(otlptranslator.UnderscoreEscapingWithoutSuffixes),
		otelprometheus.WithoutScopeInfo(),
		otelprometheus.WithoutTargetInfo(),
	)
	if err != nil {
		return nil, nil, err
	}

	return promExporter, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}), nil
}

// NewExporterOrDie creates an exporter to export metrics to Prometheus, panics if error occurs.
func NewExporterOrDie(npdo *options.NodeProblemDetectorOptions) types.Exporter {
	if npdo.PrometheusServerPort <= 0 {
		return nil
	}

	promExporter, handler, err := newExporterAndHandler()
	if err != nil {
		klog.Fatalf("Failed to create Prometheus exporter: %v", err)
	}

	// register with the global meter provider
	otelutil.AddMetricReader(promExporter)

	addr := net.JoinHostPort(npdo.PrometheusServerAddress, strconv.Itoa(npdo.PrometheusServerPort))
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", handler)
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
