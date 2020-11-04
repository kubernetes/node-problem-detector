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

package gkestackdriverexporter

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource/gcp"
	"github.com/avast/retry-go"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"go.opencensus.io/stats/view"
	"google.golang.org/api/option"
	"k8s.io/node-problem-detector/pkg/exporters"
	seconfig "k8s.io/node-problem-detector/pkg/exporters/gkestackdriver/config"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

const exporterName = "gkestackdriver"

// NPDMetricToSDMetric maps NPD metric to Stackdriver metric
var NPDMetricToSDMetric = map[metrics.MetricID]string{
	metrics.CPURunnableTaskCountID:  "kubernetes.io/node/guest/cpu/runnable_task_count",
	metrics.CPUUsageTimeID:          "kubernetes.io/node/guest/cpu/usage_time",
	metrics.DiskAvgQueueLenID:       "kubernetes.io/node/guest/disk/queue_length",
	metrics.DiskBytesUsedID:         "kubernetes.io/node/guest/disk/bytes_used",
	metrics.DiskIOTimeID:            "kubernetes.io/node/guest/disk/io_time",
	metrics.DiskMergedOpsCountID:    "kubernetes.io/node/guest/disk/merged_operation_count",
	metrics.DiskOpsBytesID:          "kubernetes.io/node/guest/disk/operation_bytes_count",
	metrics.DiskOpsCountID:          "kubernetes.io/node/guest/disk/operation_count",
	metrics.DiskOpsTimeID:           "kubernetes.io/node/guest/disk/operation_time",
	metrics.DiskWeightedIOID:        "kubernetes.io/node/guest/disk/weighted_io_time",
	metrics.HostUptimeID:            "kubernetes.io/node/guest/system/uptime",
	metrics.MemoryAnonymousUsedID:   "kubernetes.io/node/guest/memory/anonymous_used",
	metrics.MemoryBytesUsedID:       "kubernetes.io/node/guest/memory/bytes_used",
	metrics.MemoryDirtyUsedID:       "kubernetes.io/node/guest/memory/dirty_used",
	metrics.MemoryPageCacheUsedID:   "kubernetes.io/node/guest/memory/page_cache_used",
	metrics.MemoryUnevictableUsedID: "kubernetes.io/node/guest/memory/unevictable_used",
	metrics.ProblemCounterID:        "kubernetes.io/node/guest/system/problem_count",
	metrics.ProblemGaugeID:          "kubernetes.io/node/guest/system/problem_state",
}

type commandLineOptions struct {
	configPath string
}

type gkeStackdriverExporter struct {
	config seconfig.GKEStackdriverExporterConfig
}

// ExportProblems does nothing.
// Stackdriver exporter only exports metrics.
func (se *gkeStackdriverExporter) ExportProblems(status *types.Status) {
	return
}

func (clo *commandLineOptions) SetFlags(fs *pflag.FlagSet) {
	fs.StringVar(&clo.configPath, "exporter.gkestackdriver", "",
		"Configuration for Stackdriver exporter for GKE nodes. Set to config file path.")
}

func init() {
	clo := commandLineOptions{}
	exporters.Register(exporterName, types.ExporterHandler{
		CreateExporterOrDie: NewExporterOrDie,
		Options:             &clo})
}

// NewExporterOrDie creates an exporter to export metrics to Stackdriver, panics if error occurs.
func NewExporterOrDie(clo types.CommandLineOptions) types.Exporter {
	options, ok := clo.(*commandLineOptions)
	if !ok {
		glog.Fatalf("Wrong type for the command line options of Stackdriver Exporter: %s.", reflect.TypeOf(clo))
	}
	if options.configPath == "" {
		return nil
	}

	se := gkeStackdriverExporter{}

	// Apply configurations.
	f, err := ioutil.ReadFile(options.configPath)
	if err != nil {
		glog.Fatalf("Failed to read configuration file %q: %v", options.configPath, err)
	}
	err = json.Unmarshal(f, &se.config)
	if err != nil {
		glog.Fatalf("Failed to unmarshal configuration file %q: %v", options.configPath, err)
	}
	se.config.ApplyConfiguration()

	glog.Infof("Starting Stackdriver exporter %s", options.configPath)

	se.populateMetadataOrDie()
	se.setupOpenCensusViewExporterOrDie()

	return &se
}

func (se *gkeStackdriverExporter) populateMetadataOrDie() {
	if !se.config.GKEMetadata.HasMissingField() {
		glog.Infof("Using metadata specified in the config file: %+v", se.config.GKEMetadata)
		return
	}

	metadataFetchTimeout, err := time.ParseDuration(se.config.MetadataFetchTimeout)
	if err != nil {
		glog.Fatalf("Failed to parse MetadataFetchTimeout %q: %v", se.config.MetadataFetchTimeout, err)
	}

	metadataFetchInterval, err := time.ParseDuration(se.config.MetadataFetchInterval)
	if err != nil {
		glog.Fatalf("Failed to parse MetadataFetchInterval %q: %v", se.config.MetadataFetchInterval, err)
	}

	glog.Infof("Populating metadata.")
	err = retry.Do(se.config.GKEMetadata.Populate,
		retry.Delay(metadataFetchInterval),
		retry.Attempts(uint(metadataFetchTimeout/metadataFetchInterval)),
		retry.DelayType(retry.FixedDelay))
	if err == nil {
		glog.Infof("Using metadata: %+v", se.config.GKEMetadata)
		return
	}
	if se.config.PanicOnMetadataFetchFailure {
		glog.Fatalf("Failed to populate metadata: %v", err)
	} else {
		glog.Errorf("Failed to populate metadata: %v", err)
	}
}

func (se *gkeStackdriverExporter) setupOpenCensusViewExporterOrDie() {
	clientOption := option.WithEndpoint(se.config.APIEndpoint)

	var globalLabels stackdriver.Labels
	// We set these here to prevent additional metadata server calls from stackdriver.NewExporter.
	globalLabels.Set("location", se.config.GKEMetadata.Location, "The location of the VM")
	globalLabels.Set("cluster_name", se.config.GKEMetadata.ClusterName, "The name of the GKE cluster")
	globalLabels.Set("node_name", se.config.GKEMetadata.NodeName, "The name of the node")
	globalLabels.Set("os_version", se.config.GKEMetadata.OSVersion, "The OS of the VM")
	globalLabels.Set("kernel_version", se.config.GKEMetadata.KernelVersion, "The kernel version of the VM")

	viewExporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               se.config.GKEMetadata.ProjectID,
		Location:                se.config.GKEMetadata.Location,
		MonitoringClientOptions: []option.ClientOption{clientOption},
		MonitoredResource: &gcp.GKEContainer{
			ProjectID:                  se.config.GKEMetadata.ProjectID,
			InstanceID:                 se.config.GKEMetadata.InstanceID,
			ClusterName:                se.config.GKEMetadata.ClusterName,
			Zone:                       se.config.GKEMetadata.Location,
			LoggingMonitoringV2Enabled: true, // U	se "k8s_container" type
		},
		GetMetricType:           getMetricTypeConversionFunction(),
		DefaultMonitoringLabels: &globalLabels,
	})
	if err != nil {
		glog.Fatalf("Failed to create Stackdriver OpenCensus view exporter: %v", err)
	}

	exportPeriod, err := time.ParseDuration(se.config.ExportPeriod)
	if err != nil {
		glog.Fatalf("Failed to parse ExportPeriod %q: %v", se.config.ExportPeriod, err)
	}

	view.SetReportingPeriod(exportPeriod)
	view.RegisterExporter(viewExporter)
}

func getMetricTypeConversionFunction() func(*view.View) string {
	return func(view *view.View) string {
		metricID, ok := metrics.MetricMap.ViewNameToMetricID(view.Measure.Name())
		if !ok {
			return ""
		}
		stackdriverMetricType, ok := NPDMetricToSDMetric[metricID]
		if !ok {
			return ""
		}
		return stackdriverMetricType
	}
}
