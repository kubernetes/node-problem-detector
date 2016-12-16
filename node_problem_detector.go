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

	"k8s.io/node-problem-detector/pkg/kernelmonitor"
	"k8s.io/node-problem-detector/pkg/problemdetector"
	"github.com/golang/glog"
	"net/url"
)

var (
	kernelMonitorConfigPath = flag.String("kernel-monitor", "/config/kernel_monitor.json", "The path to the kernel monitor config file")
	apiServer = flag.String("apiserver", "", "URI used to connect to Kubernetes ApiServer")
)

func validateCmdParams() {
	if len(*apiServer) == 0 {
		glog.Fatal("apiserver argument is empty")
	} else if _, err := url.Parse(*apiServer); err != nil {
		glog.Fatalf("apiserver argument %q is not a valid HTTP URI. %s", *apiServer, err)
	}
}

func main() {
	flag.Parse()
	validateCmdParams()

	k := kernelmonitor.NewKernelMonitorOrDie(*kernelMonitorConfigPath)
	p := problemdetector.NewProblemDetector(k, *apiServer)
	p.Run()
}
