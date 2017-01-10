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
	"net/url"

	"k8s.io/node-problem-detector/pkg/kernelmonitor"
	"k8s.io/node-problem-detector/pkg/problemdetector"

	"github.com/golang/glog"
)

var (
	kernelMonitorConfigPath = flag.String("kernel-monitor", "/config/kernel_monitor.json", "The path to the kernel monitor config file")
	apiServerOverride       = flag.String("apiserver-override", "", "custom URI used to connect to Kubernetes ApiServer")
)

func validateCmdParams() {
	if _, err := url.Parse(*apiServerOverride); err != nil {
		glog.Fatalf("apiserver-override %q is not a valid HTTP URI: %v", *apiServerOverride, err)
	}
}

func main() {
	flag.Parse()
	validateCmdParams()

	k := kernelmonitor.NewKernelMonitorOrDie(*kernelMonitorConfigPath)
	p := problemdetector.NewProblemDetector(k, *apiServerOverride)
	p.Run()
}
