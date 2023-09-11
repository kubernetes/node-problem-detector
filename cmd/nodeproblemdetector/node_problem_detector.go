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
	"github.com/golang/glog"
	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/exporterplugins"
	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/problemdaemonplugins"
	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/exporters"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter"
	"k8s.io/node-problem-detector/pkg/exporters/prometheusexporter"
	"k8s.io/node-problem-detector/pkg/healingsync"
	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemdetector"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/version"
)

func npdInteractive(npdo *options.NodeProblemDetectorOptions) {
	termCh := make(chan error, 1)
	defer close(termCh)

	if err := npdMain(npdo, termCh); err != nil {
		glog.Fatalf("Problem detector failed with error: %v", err)
	}
}

func npdMain(npdo *options.NodeProblemDetectorOptions, termCh <-chan error) error {
	if npdo.PrintVersion {
		version.PrintVersion()
		return nil
	}

	npdo.SetNodeNameOrDie()
	npdo.SetConfigFromDeprecatedOptionsOrDie()
	npdo.ValidOrDie()

	// Initialize problem daemons.
	problemDaemonMap := problemdaemon.NewProblemDaemons(npdo.MonitorConfigPaths)
	if len(problemDaemonMap) == 0 {
		glog.Fatalf("No problem daemon is configured")
	}

	// Initialize exporters.
	defaultExporters := []types.Exporter{}
	if ke := k8sexporter.NewExporterOrDie(npdo); ke != nil {
		defaultExporters = append(defaultExporters, ke)
		glog.Info("K8s exporter started.")
	}
	if pe := prometheusexporter.NewExporterOrDie(npdo); pe != nil {
		defaultExporters = append(defaultExporters, pe)
		glog.Info("Prometheus exporter started.")
	}

	plugableExporters := exporters.NewExporters()

	npdExporters := []types.Exporter{}
	npdExporters = append(npdExporters, defaultExporters...)
	npdExporters = append(npdExporters, plugableExporters...)

	if len(npdExporters) == 0 {
		glog.Fatalf("No exporter is successfully setup")
	}

	//go controller.NewSelfHealingTaskInstanceCache()
	// Initialize cronjob
	//循环监听任务
	c := healingsync.NewCronService(problemDaemonMap, npdo.SyncInterval, npdo.SyncUrl)
	go c.Run(termCh)

	// Initialize NPD core.
	p := problemdetector.NewProblemDetector(problemDaemonMap, npdExporters)
	//c.GetChn() 获取任务的通道
	return p.Run(termCh, c.GetChn())
}
