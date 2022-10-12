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
	Run(termCh <-chan error, cron <-chan *ProblemSync) error
}

type problemDetector struct {
	monitors  map[string]types.Monitor
	exporters []types.Exporter
	channels  map[string]<-chan *types.Status
	statuses  chan *types.Status
}

// NewProblemDetector creates the problem detector. Currently we just directly passed in the problem daemons, but
// in the future we may want to let the problem daemons register themselves.
func NewProblemDetector(monitors map[string]types.Monitor, exporters []types.Exporter) ProblemDetector {
	return &problemDetector{
		monitors:  monitors,
		exporters: exporters,
		channels:  make(map[string]<-chan *types.Status),
		statuses:  make(chan *types.Status),
	}
}

// Run starts the problem detector.
func (p *problemDetector) Run(termCh <-chan error, sync <-chan *ProblemSync) error {
	// Start the log monitors one by one.
	failureCount := 0
	for name, m := range p.monitors {
		ch, err := m.Start()
		if err != nil {
			// Do not return error and keep on trying the following config files.
			glog.Errorf("Failed to start problem daemon %v: %v", m, err)
			failureCount++
			continue
		}
		if ch != nil {
			p.channels[name] = ch
		}
	}

	allMonitors := p.monitors
	if len(allMonitors) == failureCount {
		return fmt.Errorf("no problem daemon is successfully setup")
	}

	defer func() {
		for _, m := range p.monitors {
			m.Stop()
		}
	}()

	p.groupChannel()
	glog.Info("Problem detector started")

	for {
		select {
		case <-termCh:
			return nil
		case status := <-p.statuses:
			for _, exporter := range p.exporters {
				exporter.ExportProblems(status)
			}
		case task := <-sync:
			p.syncDetector(task)
		}
	}
}

func (p *problemDetector) groupChannel() {
	for _, ch := range p.channels {
		go func(c <-chan *types.Status) {
			for status := range c {
				p.statuses <- status
			}
		}(ch)
	}
}

func (p *problemDetector) syncDetector(task *ProblemSync) {
	if task == nil || task.ConfigName == "" {
		glog.Warningf("syncDetector invalid argument")
		return
	}

	if m, ok := p.monitors[task.ConfigName]; ok {
		m.Stop()
		delete(p.monitors, task.ConfigName)
		delete(p.channels, task.ConfigName)
	}

	if task.IsDelete {
		return
	}

	ch, err := task.Monitors.Start()
	if err != nil {
		glog.Errorf("Failed to start problem daemon %v: %v", task.Monitors, err)
		return
	}
	go func(c <-chan *types.Status) {
		for status := range c {
			p.statuses <- status
		}
	}(ch)

	p.monitors[task.ConfigName] = task.Monitors
	p.channels[task.ConfigName] = ch
}
