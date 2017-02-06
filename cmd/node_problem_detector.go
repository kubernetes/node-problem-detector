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
	"net/url"
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

func validateCmdParams(npdo *options.NodeProblemDetectorOptions) {
	if _, err := url.Parse(npdo.ApiServerOverride); err != nil {
		glog.Fatalf("apiserver-override %q is not a valid HTTP URI: %v", npdo.ApiServerOverride, err)
	}
}

func setNodeNameOrDie(npdo *options.NodeProblemDetectorOptions) {
	var nodeName string

	// Check hostname override first for customized node name.
	if npdo.HostnameOverride != "" {
		return
	}

	// Get node name from environment variable NODE_NAME
	// By default, assume that the NODE_NAME env should have been set with
	// downward api or user defined exported environment variable. We prefer it because sometimes
	// the hostname returned by os.Hostname is not right because:
	// 1. User may override the hostname.
	// 2. For some cloud providers, os.Hostname is different from the real hostname.
	nodeName = os.Getenv("NODE_NAME")
	if nodeName != "" {
		npdo.HostnameOverride = nodeName
		return
	}

	// For backward compatibility. If the env is not set, get the hostname
	// from os.Hostname(). This may not work for all configurations and
	// environments.
	nodeName, err := os.Hostname()
	if err != nil {
		glog.Fatalf("Failed to get host name: %v", err)
	}

	npdo.HostnameOverride = nodeName
}

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
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		glog.Fatalf("Failed to start server: %v", err)
	}
}

func main() {
	npdo := options.NewNodeProblemDetectorOptions()
	npdo.AddFlags(pflag.CommandLine)

	pflag.Parse()

	validateCmdParams(npdo)

	if npdo.PrintVersion {
		version.PrintVersion()
		os.Exit(0)
	}

	setNodeNameOrDie(npdo)

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
