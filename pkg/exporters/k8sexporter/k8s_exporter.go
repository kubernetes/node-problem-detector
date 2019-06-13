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

package k8sexporter

import (
	"net"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/clock"

	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter/condition"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter/problemclient"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
)

type k8sExporter struct {
	client           problemclient.Client
	conditionManager condition.ConditionManager
}

// NewExporterOrDie creates a exporter for Kubernetes apiserver exporting, panics if error occurs.
func NewExporterOrDie(npdo *options.NodeProblemDetectorOptions) types.Exporter {
	if !npdo.EnableK8sExporter {
		return nil
	}

	c := problemclient.NewClientOrDie(npdo)
	ke := k8sExporter{
		client:           c,
		conditionManager: condition.NewConditionManager(c, clock.RealClock{}),
	}

	ke.startHTTPReporting(npdo)
	ke.conditionManager.Start()

	return &ke
}

func (ke *k8sExporter) ExportProblems(status *types.Status) {
	for _, event := range status.Events {
		ke.client.Eventf(util.ConvertToAPIEventType(event.Severity), status.Source, event.Reason, event.Message)
	}
	for _, cdt := range status.Conditions {
		ke.conditionManager.UpdateCondition(cdt)
	}
}

func (ke *k8sExporter) startHTTPReporting(npdo *options.NodeProblemDetectorOptions) {
	if npdo.ServerPort <= 0 {
		return
	}
	mux := http.NewServeMux()

	// Add healthz http request handler. Always return ok now, add more health check
	// logic in the future.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Add the handler to serve condition http request.
	mux.HandleFunc("/conditions", func(w http.ResponseWriter, r *http.Request) {
		util.ReturnHTTPJson(w, ke.conditionManager.GetConditions())
	})

	addr := net.JoinHostPort(npdo.ServerAddress, strconv.Itoa(npdo.ServerPort))
	go func() {
		err := http.ListenAndServe(addr, mux)
		if err != nil {
			glog.Fatalf("Failed to start server: %v", err)
		}
	}()
}
