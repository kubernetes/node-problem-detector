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
	"net/http"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/util/clock"

	"k8s.io/node-problem-detector/pkg/condition"
	"k8s.io/node-problem-detector/pkg/kernelmonitor"
	"k8s.io/node-problem-detector/pkg/problemclient"
	"k8s.io/node-problem-detector/pkg/util"
)

// ProblemDetector collects statuses from all problem daemons and update the node condition and send node event.
type ProblemDetector interface {
	Run() error
	RegisterHTTPHandlers()
}

type problemDetector struct {
	client           problemclient.Client
	conditionManager condition.ConditionManager
	// TODO(random-liu): Use slices of problem daemons if multiple monitors are needed in the future
	monitor kernelmonitor.KernelMonitor
}

// NewProblemDetector creates the problem detector. Currently we just directly passed in the problem daemons, but
// in the future we may want to let the problem daemons register themselves.
func NewProblemDetector(monitor kernelmonitor.KernelMonitor, client problemclient.Client) ProblemDetector {
	return &problemDetector{
		client:           client,
		conditionManager: condition.NewConditionManager(client, clock.RealClock{}),
		monitor:          monitor,
	}
}

// Run starts the problem detector.
func (p *problemDetector) Run() error {
	p.conditionManager.Start()
	ch, err := p.monitor.Start()
	if err != nil {
		return err
	}
	glog.Info("Problem detector started")
	for {
		select {
		case status := <-ch:
			for _, event := range status.Events {
				p.client.Eventf(util.ConvertToAPIEventType(event.Severity), status.Source, event.Reason, event.Message)
			}
			for _, condition := range status.Conditions {
				p.conditionManager.UpdateCondition(condition)
			}
		}
	}
}

// RegisterHTTPHandlers registers http handlers of node problem detector.
func (p *problemDetector) RegisterHTTPHandlers() {
	// Add the handler to serve condition http request.
	http.HandleFunc("/conditions", func(w http.ResponseWriter, r *http.Request) {
		util.ReturnHTTPJson(w, p.conditionManager.GetConditions())
	})
}
