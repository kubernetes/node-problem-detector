/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package main

import (
	"context"
	"time"

	"k8s.io/klog/v2"

	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/exporterplugins"
	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/problemdaemonplugins"
	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/exporters"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter"
	"k8s.io/node-problem-detector/pkg/exporters/prometheusexporter"
	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemdetector"
	"k8s.io/node-problem-detector/pkg/problemmetrics"
	"k8s.io/node-problem-detector/pkg/types"
	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
	"k8s.io/node-problem-detector/pkg/version"
)

func npdMain(ctx context.Context, npdo *options.NodeProblemDetectorOptions) error {
	if npdo.PrintVersion {
		version.PrintVersion()
		return nil
	}

	npdo.SetNodeNameOrDie()
	npdo.SetConfigFromDeprecatedOptionsOrDie()
	npdo.ValidOrDie()

	// Initialize exporters first to set up the OpenTelemetry readers.
	defaultExporters := []types.Exporter{}
	if ke := k8sexporter.NewExporterOrDie(ctx, npdo); ke != nil {
		defaultExporters = append(defaultExporters, ke)
		klog.Info("K8s exporter started.")
	}
	if pe := prometheusexporter.NewExporterOrDie(npdo); pe != nil {
		defaultExporters = append(defaultExporters, pe)
		klog.Info("Prometheus exporter started.")
	}
	plugableExporters := exporters.NewExporters()

	// Initialize OpenTelemetry meter provider with all registered readers
	// This must be called after all exporters have been created and registered their readers
	meterProvider := otelutil.InitializeMeterProvider()
	defer func() {
		// Drop the cancellation of ctx because it is likely already canceled
		// at shutdown, which would prevent flushing pending metrics.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := meterProvider.Shutdown(shutdownCtx); err != nil {
			klog.Errorf("Failed to shut down OpenTelemetry meter provider: %v", err)
		}
	}()
	problemmetrics.InitializeGlobalProblemMetricsManager()

	// Initialize problem daemons.
	problemDaemons := problemdaemon.NewProblemDaemons(npdo.MonitorConfigPaths)
	if len(problemDaemons) == 0 {
		klog.Fatalf("No problem daemon is configured")
	}

	npdExporters := []types.Exporter{}
	npdExporters = append(npdExporters, defaultExporters...)
	npdExporters = append(npdExporters, plugableExporters...)

	if len(npdExporters) == 0 {
		klog.Fatalf("No exporter is successfully setup")
	}

	// Initialize NPD core.
	p := problemdetector.NewProblemDetector(problemDaemons, npdExporters)
	return p.Run(ctx)
}
