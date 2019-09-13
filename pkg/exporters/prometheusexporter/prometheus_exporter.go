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
	"reflect"
	"strconv"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"go.opencensus.io/stats/view"

	"k8s.io/node-problem-detector/pkg/exporters"
	"k8s.io/node-problem-detector/pkg/types"
)

func init() {
	clo := commandLineOptions{}
	exporters.Register(exporterName, types.ExporterHandler{
		CreateExporterOrDie: NewExporterOrDie,
		Options:             &clo})
}

const exporterName = "prometheus"

type prometheusExporter struct{}

// NewExporterOrDie creates an exporter to export metrics to Prometheus, panics if error occurs.
func NewExporterOrDie(clo types.CommandLineOptions) types.Exporter {
	po, ok := clo.(*commandLineOptions)
	if !ok {
		glog.Fatalf("Wrong type for the command line options of Prometheus Exporter: %s.", reflect.TypeOf(clo))
	}

	if po.PrometheusServerPort <= 0 {
		return nil
	}

	addr := net.JoinHostPort(po.PrometheusServerAddress, strconv.Itoa(po.PrometheusServerPort))
	pe, err := prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		glog.Fatalf("Failed to create Prometheus exporter: %v", err)
	}
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", pe)
		if err := http.ListenAndServe(addr, mux); err != nil {
			glog.Fatalf("Failed to start Prometheus scrape endpoint: %v", err)
		}
	}()
	view.RegisterExporter(pe)
	return &prometheusExporter{}
}

// ExportProblems does nothing.
// Prometheus exporter only exports metrics.
func (pe *prometheusExporter) ExportProblems(status *types.Status) {
	return
}

type commandLineOptions struct {
	// PrometheusServerPort is the port to bind the Prometheus scrape endpoint. Use 0 to disable.
	PrometheusServerPort int
	// PrometheusServerAddress is the address to bind the Prometheus scrape endpoint.
	PrometheusServerAddress string
}

func (clo *commandLineOptions) SetFlags(fs *pflag.FlagSet) {
	fs.IntVar(&clo.PrometheusServerPort, "prometheus-port",
		20257, "The port to bind the Prometheus scrape endpoint. Prometheus exporter is enabled by default at port 20257. Use 0 to disable.")
	fs.StringVar(&clo.PrometheusServerAddress, "prometheus-address",
		"127.0.0.1", "The address to bind the Prometheus scrape endpoint.")
}
