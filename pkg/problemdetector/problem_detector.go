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

package problemdetector

import (
	"fmt"

	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/types"
)

// ProblemDetector collects statuses from all problem daemons and update the node condition and send node event.
type ProblemDetector interface {
	Run() error
}

type problemDetector struct {
	monitors  []types.Monitor
	exporters []types.Exporter
}

// NewProblemDetector creates the problem detector. Currently we just directly passed in the problem daemons, but
// in the future we may want to let the problem daemons register themselves.
func NewProblemDetector(monitors []types.Monitor, exporters []types.Exporter) ProblemDetector {
	return &problemDetector{
		monitors:  monitors,
		exporters: exporters,
	}
}

// Run starts the problem detector.
func (p *problemDetector) Run() error {
	// Start the log monitors one by one.
	var chans []<-chan *types.Status
	failureCount := 0
	for _, m := range p.monitors {
		ch, err := m.Start()
		if err != nil {
			// Do not return error and keep on trying the following config files.
			glog.Errorf("Failed to start problem daemon %v: %v", m, err)
			failureCount += 1
			continue
		}
		if ch != nil {
			chans = append(chans, ch)
		}
	}
	if len(p.monitors) == failureCount {
		return fmt.Errorf("no problem daemon is successfully setup")
	}
	ch := groupChannel(chans)
	glog.Info("Problem detector started")

	for {
		select {
		case status := <-ch:
			for _, exporter := range p.exporters {
				exporter.ExportProblems(status)
			}
		}
	}
}

func groupChannel(chans []<-chan *types.Status) <-chan *types.Status {
	statuses := make(chan *types.Status)
	for _, ch := range chans {
		go func(c <-chan *types.Status) {
			for status := range c {
				statuses <- status
			}
		}(ch)
	}
	return statuses
}
