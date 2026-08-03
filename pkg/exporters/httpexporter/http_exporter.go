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

// Package httpexporter provides a standalone HTTP server exposing /healthz,
// /conditions and /debug/pprof without requiring a Kubernetes API server.
package httpexporter

import (
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"sync"

	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
)

type httpExporter struct {
	mu         sync.RWMutex
	conditions map[string]types.Condition
}

// NewExporterOrDie creates the standalone HTTP exporter and starts the server.
// Returns nil if --port is 0 (disabled). Panics on bind errors.
func NewExporterOrDie(npdo *options.NodeProblemDetectorOptions) types.Exporter {
	if npdo.ServerPort <= 0 {
		return nil
	}

	he := &httpExporter{
		conditions: make(map[string]types.Condition),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			klog.Errorf("Failed to write response: %v", err)
		}
	})

	mux.HandleFunc("/conditions", func(w http.ResponseWriter, r *http.Request) {
		util.ReturnHTTPJson(w, he.getConditions())
	})

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	addr := net.JoinHostPort(npdo.ServerAddress, strconv.Itoa(npdo.ServerPort))
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			klog.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	klog.Infof("HTTP exporter started on %s", addr)
	return he
}

// ExportProblems updates the in-memory condition store from the incoming status.
func (he *httpExporter) ExportProblems(status *types.Status) {
	he.mu.Lock()
	defer he.mu.Unlock()
	for _, cdt := range status.Conditions {
		he.conditions[cdt.Type] = cdt
	}
}

func (he *httpExporter) getConditions() []types.Condition {
	he.mu.RLock()
	defer he.mu.RUnlock()
	conditions := make([]types.Condition, 0, len(he.conditions))
	for _, c := range he.conditions {
		conditions = append(conditions, c)
	}
	return conditions
}
