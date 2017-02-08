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
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"k8s.io/node-problem-detector/pkg/kernelmonitor"
	"k8s.io/node-problem-detector/pkg/options"
	"k8s.io/node-problem-detector/pkg/problemclient"
	"k8s.io/node-problem-detector/pkg/problemdetector"
	"k8s.io/node-problem-detector/pkg/version"
)

func startHTTPServer(p problemdetector.ProblemDetector, npdo *options.NodeProblemDetectorOptions) {
	// Add healthz http request handler. Always return ok now, add more health check
	// logic in the future.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// Add the http handlers in problem detector.
	p.RegisterHTTPHandlers()

	addr := net.JoinHostPort(npdo.ServerAddress, strconv.Itoa(npdo.ServerPort))
	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			glog.Fatalf("Failed to start server: %v", err)
		}
	}()
}

func main() {
	npdo := options.NewNodeProblemDetectorOptions()
	npdo.AddFlags(pflag.CommandLine)

	pflag.Parse()

	npdo.SetNodeNameOrDie()

	npdo.ValidOrDie()

	if npdo.PrintVersion {
		version.PrintVersion()
		os.Exit(0)
	}

	k := kernelmonitor.NewKernelMonitorOrDie(npdo.KernelMonitorConfigPath)
	c := problemclient.NewClientOrDie(npdo)
	p := problemdetector.NewProblemDetector(k, c)

	// Start http server.
	if npdo.ServerPort > 0 {
		startHTTPServer(p, npdo)
	}

	if err := p.Run(); err != nil {
		glog.Fatalf("Problem detector failed with error: %v", err)
	}
}
