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

package stackdriverexporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"time"

	gcpmetric "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"github.com/avast/retry-go/v4"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/pkg/exporters"
	seconfig "k8s.io/node-problem-detector/pkg/exporters/stackdriver/config"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
	otelutil "k8s.io/node-problem-detector/pkg/util/otel"
)

func init() {
	clo := commandLineOptions{}
	exporters.Register(exporterName, types.ExporterHandler{
		CreateExporterOrDie: NewExporterOrDie,
		Options:             &clo,
	})
}

const exporterName = "stackdriver"

var NPDMetricToSDMetric = map[metrics.MetricID]string{
	metrics.CPURunnableTaskCountID:  "compute.googleapis.com/guest/cpu/runnable_task_count",
	metrics.CPUUsageTimeID:          "compute.googleapis.com/guest/cpu/usage_time",
	metrics.CPULoad1m:               "compute.googleapis.com/guest/cpu/load_1m",
	metrics.CPULoad5m:               "compute.googleapis.com/guest/cpu/load_5m",
	metrics.CPULoad15m:              "compute.googleapis.com/guest/cpu/load_15m",
	metrics.DiskAvgQueueLenID:       "compute.googleapis.com/guest/disk/queue_length",
	metrics.DiskBytesUsedID:         "compute.googleapis.com/guest/disk/bytes_used",
	metrics.DiskPercentUsedID:       "compute.googleapis.com/guest/disk/percent_used",
	metrics.DiskIOTimeID:            "compute.googleapis.com/guest/disk/io_time",
	metrics.DiskMergedOpsCountID:    "compute.googleapis.com/guest/disk/merged_operation_count",
	metrics.DiskOpsBytesID:          "compute.googleapis.com/guest/disk/operation_bytes_count",
	metrics.DiskOpsCountID:          "compute.googleapis.com/guest/disk/operation_count",
	metrics.DiskOpsTimeID:           "compute.googleapis.com/guest/disk/operation_time",
	metrics.DiskWeightedIOID:        "compute.googleapis.com/guest/disk/weighted_io_time",
	metrics.HostUptimeID:            "compute.googleapis.com/guest/system/uptime",
	metrics.MemoryAnonymousUsedID:   "compute.googleapis.com/guest/memory/anonymous_used",
	metrics.MemoryBytesUsedID:       "compute.googleapis.com/guest/memory/bytes_used",
	metrics.MemoryDirtyUsedID:       "compute.googleapis.com/guest/memory/dirty_used",
	metrics.MemoryPageCacheUsedID:   "compute.googleapis.com/guest/memory/page_cache_used",
	metrics.MemoryUnevictableUsedID: "compute.googleapis.com/guest/memory/unevictable_used",
	metrics.MemoryPercentUsedID:     "compute.googleapis.com/guest/memory/percent_used",
	metrics.ProblemCounterID:        "compute.googleapis.com/guest/system/problem_count",
	metrics.ProblemGaugeID:          "compute.googleapis.com/guest/system/problem_state",
	metrics.OSFeatureID:             "compute.googleapis.com/guest/system/os_feature_enabled",
	metrics.SystemProcessesTotal:    "kubernetes.io/internal/node/guest/system/processes_total",
	metrics.SystemProcsRunning:      "kubernetes.io/internal/node/guest/system/procs_running",
	metrics.SystemProcsBlocked:      "kubernetes.io/internal/node/guest/system/procs_blocked",
	metrics.SystemInterruptsTotal:   "kubernetes.io/internal/node/guest/system/interrupts_total",
	metrics.SystemCPUStat:           "kubernetes.io/internal/node/guest/system/cpu_stat",
	metrics.NetDevRxBytes:           "kubernetes.io/internal/node/guest/net/rx_bytes",
	metrics.NetDevRxPackets:         "kubernetes.io/internal/node/guest/net/rx_packets",
	metrics.NetDevRxErrors:          "kubernetes.io/internal/node/guest/net/rx_errors",
	metrics.NetDevRxDropped:         "kubernetes.io/internal/node/guest/net/rx_dropped",
	metrics.NetDevRxFifo:            "kubernetes.io/internal/node/guest/net/rx_fifo",
	metrics.NetDevRxFrame:           "kubernetes.io/internal/node/guest/net/rx_frame",
	metrics.NetDevRxCompressed:      "kubernetes.io/internal/node/guest/net/rx_compressed",
	metrics.NetDevRxMulticast:       "kubernetes.io/internal/node/guest/net/rx_multicast",
	metrics.NetDevTxBytes:           "kubernetes.io/internal/node/guest/net/tx_bytes",
	metrics.NetDevTxPackets:         "kubernetes.io/internal/node/guest/net/tx_packets",
	metrics.NetDevTxErrors:          "kubernetes.io/internal/node/guest/net/tx_errors",
	metrics.NetDevTxDropped:         "kubernetes.io/internal/node/guest/net/tx_dropped",
	metrics.NetDevTxFifo:            "kubernetes.io/internal/node/guest/net/tx_fifo",
	metrics.NetDevTxCollisions:      "kubernetes.io/internal/node/guest/net/tx_collisions",
	metrics.NetDevTxCarrier:         "kubernetes.io/internal/node/guest/net/tx_carrier",
	metrics.NetDevTxCompressed:      "kubernetes.io/internal/node/guest/net/tx_compressed",
}

func getMetricTypeConversionFunction(customMetricPrefix string) func(string) string {
	return func(metricName string) string {
		fallbackMetricType := ""
		if customMetricPrefix != "" {
			// Example fallbackMetricType: custom.googleapis.com/npd/host/uptime
			fallbackMetricType = filepath.Join(customMetricPrefix, metricName)
		}

		metricID, ok := metrics.MetricMap.ViewNameToMetricID(metricName)
		if !ok {
			return fallbackMetricType
		}
		stackdriverMetricType, ok := NPDMetricToSDMetric[metricID]
		if !ok {
			return fallbackMetricType
		}
		return stackdriverMetricType
	}
}

type stackdriverExporter struct {
	config seconfig.StackdriverExporterConfig
}

func (se *stackdriverExporter) setupOTelExporterOrDie() {
	// Create Google Cloud Monitoring exporter
	gcpExporter, err := gcpmetric.New(
		gcpmetric.WithProjectID(se.config.GCEMetadata.ProjectID),
		gcpmetric.WithMetricDescriptorTypeFormatter(se.getMetricTypeFormatter()),
	)
	if err != nil {
		klog.Fatalf("Failed to create Google Cloud Monitoring exporter: %v", err)
	}

	exportPeriod, err := time.ParseDuration(se.config.ExportPeriod)
	if err != nil {
		klog.Fatalf("Failed to parse ExportPeriod %q: %v", se.config.ExportPeriod, err)
	}

	reader := metric.NewPeriodicReader(
		gcpExporter,
		metric.WithInterval(exportPeriod),
	)

	// register with the global meter provider
	otelutil.AddMetricReader(reader)

	klog.Infof("Google Cloud Monitoring exporter configured for project %s", se.config.GCEMetadata.ProjectID)
}

// getMetricTypeFormatter returns a function to convert metrics to GCP metric types
func (se *stackdriverExporter) getMetricTypeFormatter() func(metricdata.Metrics) string {
	converter := getMetricTypeConversionFunction(se.config.CustomMetricPrefix)
	return func(m metricdata.Metrics) string {
		return converter(m.Name)
	}
}

func (se *stackdriverExporter) populateMetadataOrDie() {
	if !se.config.GCEMetadata.HasMissingField() {
		klog.Infof("Using GCE metadata specified in the config file: %+v", se.config.GCEMetadata)
		return
	}

	metadataFetchTimeout, err := time.ParseDuration(se.config.MetadataFetchTimeout)
	if err != nil {
		klog.Fatalf("Failed to parse MetadataFetchTimeout %q: %v", se.config.MetadataFetchTimeout, err)
	}

	metadataFetchInterval, err := time.ParseDuration(se.config.MetadataFetchInterval)
	if err != nil {
		klog.Fatalf("Failed to parse MetadataFetchInterval %q: %v", se.config.MetadataFetchInterval, err)
	}

	klog.Infof("Populating GCE metadata by querying GCE metadata server.")
	err = retry.Do(se.config.GCEMetadata.PopulateFromGCE,
		retry.Delay(metadataFetchInterval),
		retry.Attempts(uint(metadataFetchTimeout/metadataFetchInterval)),
		retry.DelayType(retry.FixedDelay))
	if err == nil {
		klog.Infof("Using GCE metadata: %+v", se.config.GCEMetadata)
		return
	}
	if se.config.PanicOnMetadataFetchFailure {
		klog.Fatalf("Failed to populate GCE metadata: %v", err)
	} else {
		klog.Errorf("Failed to populate GCE metadata: %v", err)
	}
}

// ExportProblems does nothing.
// Stackdriver exporter only exports metrics.
func (se *stackdriverExporter) ExportProblems(status *types.Status) {
}

type commandLineOptions struct {
	configPath string
}

func (clo *commandLineOptions) SetFlags(fs *pflag.FlagSet) {
	fs.StringVar(&clo.configPath, "exporter.stackdriver", "",
		"Configuration for Stackdriver exporter. Set to config file path.")
}

// NewExporterOrDie creates an exporter to export metrics to Stackdriver, panics if error occurs.
func NewExporterOrDie(clo types.CommandLineOptions) types.Exporter {
	options, ok := clo.(*commandLineOptions)
	if !ok {
		klog.Fatalf("Wrong type for the command line options of Stackdriver Exporter: %s.", reflect.TypeOf(clo))
	}
	if options.configPath == "" {
		return nil
	}

	se := stackdriverExporter{}

	// Apply configurations.
	f, err := os.ReadFile(options.configPath)
	if err != nil {
		klog.Fatalf("Failed to read configuration file %q: %v", options.configPath, err)
	}
	err = json.Unmarshal(f, &se.config)
	if err != nil {
		klog.Fatalf("Failed to unmarshal configuration file %q: %v", options.configPath, err)
	}
	se.config.ApplyConfiguration()

	klog.Infof("Starting Stackdriver exporter %s", options.configPath)

	se.populateMetadataOrDie()
	se.setupOTelExporterOrDie()

	return &se
}
