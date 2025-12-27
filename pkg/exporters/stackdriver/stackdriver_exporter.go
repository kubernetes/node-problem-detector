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
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"time"

	gcpmetricexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"github.com/avast/retry-go/v4"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/api/option"
	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/pkg/exporters"
	seconfig "k8s.io/node-problem-detector/pkg/exporters/stackdriver/config"
	"k8s.io/node-problem-detector/pkg/types"
	npdmetrics "k8s.io/node-problem-detector/pkg/util/metrics"
)

func init() {
	clo := commandLineOptions{}
	exporters.Register(exporterName, types.ExporterHandler{
		CreateExporterOrDie: NewExporterOrDie,
		Options:             &clo,
	})
}

const exporterName = "stackdriver"

var NPDMetricToSDMetric = map[npdmetrics.MetricID]string{
	npdmetrics.CPURunnableTaskCountID:  "compute.googleapis.com/guest/cpu/runnable_task_count",
	npdmetrics.CPUUsageTimeID:          "compute.googleapis.com/guest/cpu/usage_time",
	npdmetrics.CPULoad1m:               "compute.googleapis.com/guest/cpu/load_1m",
	npdmetrics.CPULoad5m:               "compute.googleapis.com/guest/cpu/load_5m",
	npdmetrics.CPULoad15m:              "compute.googleapis.com/guest/cpu/load_15m",
	npdmetrics.DiskAvgQueueLenID:       "compute.googleapis.com/guest/disk/queue_length",
	npdmetrics.DiskBytesUsedID:         "compute.googleapis.com/guest/disk/bytes_used",
	npdmetrics.DiskPercentUsedID:       "compute.googleapis.com/guest/disk/percent_used",
	npdmetrics.DiskIOTimeID:            "compute.googleapis.com/guest/disk/io_time",
	npdmetrics.DiskMergedOpsCountID:    "compute.googleapis.com/guest/disk/merged_operation_count",
	npdmetrics.DiskOpsBytesID:          "compute.googleapis.com/guest/disk/operation_bytes_count",
	npdmetrics.DiskOpsCountID:          "compute.googleapis.com/guest/disk/operation_count",
	npdmetrics.DiskOpsTimeID:           "compute.googleapis.com/guest/disk/operation_time",
	npdmetrics.DiskWeightedIOID:        "compute.googleapis.com/guest/disk/weighted_io_time",
	npdmetrics.HostUptimeID:            "compute.googleapis.com/guest/system/uptime",
	npdmetrics.MemoryAnonymousUsedID:   "compute.googleapis.com/guest/memory/anonymous_used",
	npdmetrics.MemoryBytesUsedID:       "compute.googleapis.com/guest/memory/bytes_used",
	npdmetrics.MemoryDirtyUsedID:       "compute.googleapis.com/guest/memory/dirty_used",
	npdmetrics.MemoryPageCacheUsedID:   "compute.googleapis.com/guest/memory/page_cache_used",
	npdmetrics.MemoryUnevictableUsedID: "compute.googleapis.com/guest/memory/unevictable_used",
	npdmetrics.MemoryPercentUsedID:     "compute.googleapis.com/guest/memory/percent_used",
	npdmetrics.ProblemCounterID:        "compute.googleapis.com/guest/system/problem_count",
	npdmetrics.ProblemGaugeID:          "compute.googleapis.com/guest/system/problem_state",
	npdmetrics.OSFeatureID:             "compute.googleapis.com/guest/system/os_feature_enabled",
	npdmetrics.SystemProcessesTotal:    "kubernetes.io/internal/node/guest/system/processes_total",
	npdmetrics.SystemProcsRunning:      "kubernetes.io/internal/node/guest/system/procs_running",
	npdmetrics.SystemProcsBlocked:      "kubernetes.io/internal/node/guest/system/procs_blocked",
	npdmetrics.SystemInterruptsTotal:   "kubernetes.io/internal/node/guest/system/interrupts_total",
	npdmetrics.SystemCPUStat:           "kubernetes.io/internal/node/guest/system/cpu_stat",
	npdmetrics.NetDevRxBytes:           "kubernetes.io/internal/node/guest/net/rx_bytes",
	npdmetrics.NetDevRxPackets:         "kubernetes.io/internal/node/guest/net/rx_packets",
	npdmetrics.NetDevRxErrors:          "kubernetes.io/internal/node/guest/net/rx_errors",
	npdmetrics.NetDevRxDropped:         "kubernetes.io/internal/node/guest/net/rx_dropped",
	npdmetrics.NetDevRxFifo:            "kubernetes.io/internal/node/guest/net/rx_fifo",
	npdmetrics.NetDevRxFrame:           "kubernetes.io/internal/node/guest/net/rx_frame",
	npdmetrics.NetDevRxCompressed:      "kubernetes.io/internal/node/guest/net/rx_compressed",
	npdmetrics.NetDevRxMulticast:       "kubernetes.io/internal/node/guest/net/rx_multicast",
	npdmetrics.NetDevTxBytes:           "kubernetes.io/internal/node/guest/net/tx_bytes",
	npdmetrics.NetDevTxPackets:         "kubernetes.io/internal/node/guest/net/tx_packets",
	npdmetrics.NetDevTxErrors:          "kubernetes.io/internal/node/guest/net/tx_errors",
	npdmetrics.NetDevTxDropped:         "kubernetes.io/internal/node/guest/net/tx_dropped",
	npdmetrics.NetDevTxFifo:            "kubernetes.io/internal/node/guest/net/tx_fifo",
	npdmetrics.NetDevTxCollisions:      "kubernetes.io/internal/node/guest/net/tx_collisions",
	npdmetrics.NetDevTxCarrier:         "kubernetes.io/internal/node/guest/net/tx_carrier",
	npdmetrics.NetDevTxCompressed:      "kubernetes.io/internal/node/guest/net/tx_compressed",
}

// getMetricDescriptorTypeFunc returns a function that maps metric names to GCP metric types.
func getMetricDescriptorTypeFunc(customMetricPrefix string) func(m metricdata.Metrics) string {
	return func(m metricdata.Metrics) string {
		viewName := m.Name

		fallbackMetricType := ""
		if customMetricPrefix != "" {
			// Example fallbackMetricType: custom.googleapis.com/npd/host/uptime
			fallbackMetricType = filepath.Join(customMetricPrefix, viewName)
		}

		metricID, ok := npdmetrics.MetricMap.ViewNameToMetricID(viewName)
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

func (se *stackdriverExporter) setupOpenTelemetryExporterOrDie() {
	clientOption := option.WithEndpoint(se.config.APIEndpoint)

	exportPeriod, err := time.ParseDuration(se.config.ExportPeriod)
	if err != nil {
		klog.Fatalf("Failed to parse ExportPeriod %q: %v", se.config.ExportPeriod, err)
	}

	// Create a resource with GCE instance information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.CloudProviderGCP,
			semconv.CloudPlatformGCPComputeEngine,
			semconv.CloudAccountID(se.config.GCEMetadata.ProjectID),
			semconv.CloudAvailabilityZone(se.config.GCEMetadata.Zone),
			semconv.HostID(se.config.GCEMetadata.InstanceID),
			semconv.HostName(se.config.GCEMetadata.InstanceName),
			attribute.String("instance_name", se.config.GCEMetadata.InstanceName),
		),
	)
	if err != nil {
		klog.Fatalf("Failed to create OpenTelemetry resource: %v", err)
	}

	exporter, err := gcpmetricexporter.New(
		gcpmetricexporter.WithProjectID(se.config.GCEMetadata.ProjectID),
		gcpmetricexporter.WithMonitoringClientOptions(clientOption),
		gcpmetricexporter.WithMetricDescriptorTypeFormatter(getMetricDescriptorTypeFunc(se.config.CustomMetricPrefix)),
	)
	if err != nil {
		klog.Fatalf("Failed to create Google Cloud Monitoring exporter: %v", err)
	}

	reader := metric.NewPeriodicReader(exporter,
		metric.WithInterval(exportPeriod),
	)

	// Add this reader to the global metrics provider with the resource
	npdmetrics.AddReaderWithResource(reader, res)
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
	se.setupOpenTelemetryExporterOrDie()

	return &se
}
