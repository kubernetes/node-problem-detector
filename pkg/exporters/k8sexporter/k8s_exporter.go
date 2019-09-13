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
	"reflect"
	"strconv"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/node-problem-detector/pkg/exporters"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter/condition"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter/options"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter/problemclient"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
)

func init() {
	clo := options.CommandLineOptions{}
	exporters.Register(exporterName, types.ExporterHandler{
		CreateExporterOrDie: NewExporterOrDie,
		Options:             &clo})
}

const exporterName = "kubernetes"

type k8sExporter struct {
	client           problemclient.Client
	conditionManager condition.ConditionManager
}

// NewExporterOrDie creates a exporter for Kubernetes apiserver exporting,
// panics if error occurs.
//
// Note that this function may be blocked (until a timeout occurs) before
// kube-apiserver becomes ready.
func NewExporterOrDie(clo types.CommandLineOptions) types.Exporter {
	k8sOptions, ok := clo.(*options.CommandLineOptions)
	if !ok {
		glog.Fatalf("Wrong type for the command line options of Kubernetes Exporter: %s.", reflect.TypeOf(clo))
	}

	if !k8sOptions.EnableK8sExporter {
		return nil
	}

	c := problemclient.NewClientOrDie(k8sOptions)

	glog.Infof("Waiting for kube-apiserver to be ready (timeout %v)...", k8sOptions.APIServerWaitTimeout)
	if err := waitForAPIServerReadyWithTimeout(c, k8sOptions); err != nil {
		glog.Warningf("kube-apiserver did not become ready: timed out on waiting for kube-apiserver to return the node object: %v", err)
	}

	ke := k8sExporter{
		client:           c,
		conditionManager: condition.NewConditionManager(c, clock.RealClock{}, k8sOptions.K8sExporterHeartbeatPeriod),
	}

	ke.startHTTPReporting(k8sOptions)
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

func (ke *k8sExporter) startHTTPReporting(k8sOptions *options.CommandLineOptions) {
	if k8sOptions.ServerPort <= 0 {
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

	addr := net.JoinHostPort(k8sOptions.ServerAddress, strconv.Itoa(k8sOptions.ServerPort))
	go func() {
		err := http.ListenAndServe(addr, mux)
		if err != nil {
			glog.Fatalf("Failed to start server: %v", err)
		}
	}()
}

func waitForAPIServerReadyWithTimeout(c problemclient.Client, k8sOptions *options.CommandLineOptions) error {
	return wait.PollImmediate(k8sOptions.APIServerWaitInterval, k8sOptions.APIServerWaitTimeout, func() (done bool, err error) {
		// If NPD can get the node object from kube-apiserver, the server is
		// ready and the RBAC permission is set correctly.
		if _, err := c.GetNode(); err == nil {
			return true, nil
		}
		return false, nil
	})
}
