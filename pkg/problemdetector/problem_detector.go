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
	"net/http"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/util/clock"

	"k8s.io/node-problem-detector/pkg/condition"
	"k8s.io/node-problem-detector/pkg/problemclient"
	"k8s.io/node-problem-detector/pkg/types"
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
	monitors         map[string]types.Monitor
}

// NewProblemDetector creates the problem detector. Currently we just directly passed in the problem daemons, but
// in the future we may want to let the problem daemons register themselves.
func NewProblemDetector(monitors map[string]types.Monitor, client problemclient.Client) ProblemDetector {
	return &problemDetector{
		client:           client,
		conditionManager: condition.NewConditionManager(client, clock.RealClock{}),
		monitors:         monitors,
	}
}

// Run starts the problem detector.
func (p *problemDetector) Run() error {
	p.conditionManager.Start()
	// Start the log monitors one by one.
	var chans []<-chan *types.Status
	for cfg, m := range p.monitors {
		ch, err := m.Start()
		if err != nil {
			// Do not return error and keep on trying the following config files.
			glog.Errorf("Failed to start log monitor %q: %v", cfg, err)
			continue
		}
		chans = append(chans, ch)
	}
	if len(chans) == 0 {
		return fmt.Errorf("no log montior is successfully setup")
	}
	ch := groupChannel(chans)
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
