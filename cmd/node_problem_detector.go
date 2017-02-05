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
	"flag"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"k8s.io/node-problem-detector/pkg/kernelmonitor"
	"k8s.io/node-problem-detector/pkg/problemdetector"
	"k8s.io/node-problem-detector/pkg/version"
)

// TODO: Move flags to options directory.
var (
	kernelMonitorConfigPath = pflag.String("kernel-monitor", "/config/kernel-monitor.json", "The path to the kernel monitor config file")
	apiServerOverride       = pflag.String("apiserver-override", "", "Custom URI used to connect to Kubernetes ApiServer")
	printVersion            = pflag.Bool("version", false, "Print version information and quit")
	hostnameOverride        = pflag.String("hostname-override", "", "Custom node name used to override hostname")
	serverPort              = pflag.Int("port", 10256, "The port to bind the node problem detector server. Use 0 to disable.")
	serverAddress           = pflag.String("address", "127.0.0.1", "The address to bind the node problem detector server.")
)

func validateCmdParams() {
	if _, err := url.Parse(*apiServerOverride); err != nil {
		glog.Fatalf("apiserver-override %q is not a valid HTTP URI: %v", *apiServerOverride, err)
	}
}

func getNodeNameOrDie() string {
	var nodeName string

	// Check hostname override first for customized node name.
	if *hostnameOverride != "" {
		return *hostnameOverride
	}

	// Get node name from environment variable NODE_NAME
	// By default, assume that the NODE_NAME env should have been set with
	// downward api or user defined exported environment variable. We prefer it because sometimes
	// the hostname returned by os.Hostname is not right because:
	// 1. User may override the hostname.
	// 2. For some cloud providers, os.Hostname is different from the real hostname.
	nodeName = os.Getenv("NODE_NAME")
	if nodeName != "" {
		return nodeName
	}

	// For backward compatibility. If the env is not set, get the hostname
	// from os.Hostname(). This may not work for all configurations and
	// environments.
	nodeName, err := os.Hostname()
	if err != nil {
		glog.Fatalf("Failed to get host name: %v", err)
	}

	return nodeName
}

func startHTTPServer(p problemdetector.ProblemDetector) {
	// Add healthz http request handler. Always return ok now, add more health check
	// logic in the future.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// Add the http handlers in problem detector.
	p.RegisterHTTPHandlers()
	err := http.ListenAndServe(net.JoinHostPort(*serverAddress, strconv.Itoa(*serverPort)), nil)
	if err != nil {
		glog.Fatalf("Failed to start server: %v", err)
	}
}

func main() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	validateCmdParams()

	if *printVersion {
		version.PrintVersion()
		os.Exit(0)
	}

	nodeName := getNodeNameOrDie()

	k := kernelmonitor.NewKernelMonitorOrDie(*kernelMonitorConfigPath)
	p := problemdetector.NewProblemDetector(k, *apiServerOverride, nodeName)

	// Start http server.
	if *serverPort > 0 {
		startHTTPServer(p)
	}

	if err := p.Run(); err != nil {
		glog.Fatalf("Problem detector failed with error: %v", err)
	}
}
