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
	"io/ioutil"
	"path/filepath"
	"reflect"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	monitoredres "contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"go.opencensus.io/stats/view"
	"google.golang.org/api/option"

	"github.com/avast/retry-go"
	"k8s.io/node-problem-detector/pkg/exporters"
	seconfig "k8s.io/node-problem-detector/pkg/exporters/stackdriver/config"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

func init() {
	clo := commandLineOptions{}
	exporters.Register(exporterName, types.ExporterHandler{
		CreateExporterOrDie: NewExporterOrDie,
		Options:             &clo})
}

const exporterName = "stackdriver"

var NPDMetricToSDMetric = map[metrics.MetricID]string{
	metrics.HostUptimeID:      "compute.googleapis.com/guest/system/uptime",
	metrics.ProblemCounterID:  "compute.googleapis.com/guest/system/problem_count",
	metrics.ProblemGaugeID:    "compute.googleapis.com/guest/system/problem_state",
	metrics.DiskAvgQueueLenID: "compute.googleapis.com/guest/disk/queue_length",
	metrics.DiskIOTimeID:      "compute.googleapis.com/guest/disk/io_time",
	metrics.DiskWeightedIOID:  "compute.googleapis.com/guest/disk/weighted_io_time",
}

func getMetricTypeConversionFunction(customMetricPrefix string) func(*view.View) string {

	return func(view *view.View) string {
		viewName := view.Measure.Name()

		fallbackMetricType := ""
		if customMetricPrefix != "" {
			// Example fallbackMetricType: custom.googleapis.com/npd/host/uptime
			fallbackMetricType = filepath.Join(customMetricPrefix, viewName)
		}

		metricID, ok := metrics.MetricMap.ViewNameToMetricID(viewName)
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

func (se *stackdriverExporter) setupOpenCensusViewExporterOrDie() {
	clientOption := option.WithEndpoint(se.config.APIEndpoint)

	var globalLabels stackdriver.Labels
	globalLabels.Set("instance_name", se.config.GCEMetadata.InstanceName, "The name of the VM instance")

	viewExporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               se.config.GCEMetadata.ProjectID,
		MonitoringClientOptions: []option.ClientOption{clientOption},
		MonitoredResource: &monitoredres.GCEInstance{
			ProjectID:  se.config.GCEMetadata.ProjectID,
			InstanceID: se.config.GCEMetadata.InstanceID,
			Zone:       se.config.GCEMetadata.Zone,
		},
		GetMetricType:           getMetricTypeConversionFunction(se.config.CustomMetricPrefix),
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

func (se *stackdriverExporter) populateMetadataOrDie() {
	if !se.config.GCEMetadata.HasMissingField() {
		glog.Infof("Using GCE metadata specified in the config file: %+v", se.config.GCEMetadata)
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

	glog.Infof("Populating GCE metadata by querying GCE metadata server.")
	err = retry.Do(se.config.GCEMetadata.PopulateFromGCE,
		retry.Delay(metadataFetchInterval),
		retry.Attempts(uint(metadataFetchTimeout/metadataFetchInterval)),
		retry.DelayType(retry.FixedDelay))
	if err == nil {
		glog.Infof("Using GCE metadata: %+v", se.config.GCEMetadata)
		return
	}
	if se.config.PanicOnMetadataFetchFailure {
		glog.Fatalf("Failed to populate GCE metadata: %v", err)
	} else {
		glog.Errorf("Failed to populate GCE metadata: %v", err)
	}
}

// ExportProblems does nothing.
// Stackdriver exporter only exports metrics.
func (se *stackdriverExporter) ExportProblems(status *types.Status) {
	return
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
		glog.Fatalf("Wrong type for the command line options of Stackdriver Exporter: %s.", reflect.TypeOf(clo))
	}
	if options.configPath == "" {
		return nil
	}

	se := stackdriverExporter{}

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
