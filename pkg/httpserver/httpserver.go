/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

// Package httpserver serves NPD's diagnostic endpoints: /healthz, /conditions
// and /debug/pprof. The server is controlled by the --port flag alone, so it
// is available whether or not any particular exporter is enabled.
package httpserver

import (
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"

	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
)

// ConditionsGetter reports the node conditions NPD currently exports. The
// Kubernetes exporter satisfies it; when that exporter is disabled there are
// no conditions to report and the server runs without a getter.
type ConditionsGetter interface {
	GetConditions() []types.Condition
}

// NewHandler builds the handler for the diagnostic endpoints. A nil getter
// serves no conditions, which is the same response as an enabled exporter
// that has not recorded any condition yet.
func NewHandler(getter ConditionsGetter) http.Handler {
	mux := http.NewServeMux()

	// Add healthz http request handler. Always return ok now, add more health check
	// logic in the future.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			klog.Errorf("Failed to write response: %v", err)
		}
	})

	// Add the handler to serve condition http request.
	mux.HandleFunc("/conditions", func(w http.ResponseWriter, r *http.Request) {
		var conditions []types.Condition
		if getter != nil {
			conditions = getter.GetConditions()
		}
		util.ReturnHTTPJson(w, conditions)
	})

	// register pprof
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return mux
}

// Start serves the diagnostic endpoints in a new goroutine. A non-positive
// npdo.ServerPort disables the server.
func Start(npdo *options.NodeProblemDetectorOptions, getter ConditionsGetter) {
	if npdo.ServerPort <= 0 {
		return
	}

	handler := NewHandler(getter)
	addr := net.JoinHostPort(npdo.ServerAddress, strconv.Itoa(npdo.ServerPort))
	go func() {
		err := http.ListenAndServe(addr, handler)
		if err != nil {
			klog.Fatalf("Failed to start server: %v", err)
		}
	}()
}
