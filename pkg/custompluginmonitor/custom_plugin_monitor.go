/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

package custompluginmonitor

import (
	"encoding/json"
	"io/ioutil"
	"time"

	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/types"

	"github.com/golang/glog"
	"k8s.io/node-problem-detector/pkg/custompluginmonitor/plugin"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

type customPluginMonitor struct {
	config     cpmtypes.CustomPluginConfig
	conditions []types.Condition
	plugin     *plugin.Plugin
	resultChan <-chan cpmtypes.Result
	statusChan chan *types.Status
	tomb       *tomb.Tomb
}

// NewCustomPluginMonitorOrDie create a new customPluginMonitor, panic if error occurs.
func NewCustomPluginMonitorOrDie(configPath string) types.Monitor {
	l := &customPluginMonitor{
		tomb: tomb.NewTomb(),
	}
	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		glog.Fatalf("Failed to read configuration file %q: %v", configPath, err)
	}
	err = json.Unmarshal(f, &l.config)
	if err != nil {
		glog.Fatalf("Failed to unmarshal configuration file %q: %v", configPath, err)
	}
	// Apply configurations
	err = (&l.config).ApplyConfiguration()
	if err != nil {
		glog.Fatalf("Failed to apply configuration for %q. %v", configPath, err)
	}

	// Validate configurations
	err = l.config.Validate()
	if err != nil {
		glog.Fatalf("Failed to validate custom plugin config %+v. %v", l.config, err)
	}

	glog.Infof("Finish parsing custom plugin monitor config file: %+v", l.config)

	l.plugin = plugin.NewPlugin(l.config)
	// A 1000 size channel should be big enough.
	l.statusChan = make(chan *types.Status, 1000)
	return l
}

func (l *customPluginMonitor) Start() (<-chan *types.Status, error) {
	glog.Info("Start custom plugin monitor")
	go l.plugin.Run()
	go l.monitorLoop()
	return l.statusChan, nil
}

func (l *customPluginMonitor) Stop() {
	glog.Info("Stop custom plugin monitor")
	l.tomb.Stop()
}

// monitorLoop is the main loop of log monitor.
func (l *customPluginMonitor) monitorLoop() {
	l.initializeStatus()

	resultChan := l.plugin.GetResultChan()

	for {
		select {
		case result := <-resultChan:
			glog.V(3).Infof("Receive new plugin result. %+v", result)
			status := l.generateStatus(result)
			glog.Infof("New status generated: %+v", status)
			l.statusChan <- status
		case <-l.tomb.Stopping():
			l.plugin.Stop()
			glog.Infof("Custom plugin monitor stopped")
			l.tomb.Done()
			break
		}
	}
}

// generateStatus generates status from the plugin check result.
func (l *customPluginMonitor) generateStatus(result cpmtypes.Result) *types.Status {
	timestamp := time.Now()
	var events []types.Event
	if result.Rule.Type == types.Temp {
		// For temporary error only generate event when exit status is above warning
		if result.ExitStatus >= cpmtypes.NonOK {
			events = append(events, types.Event{
				Severity:  types.Warn,
				Timestamp: timestamp,
				Reason:    result.Rule.Reason,
				Message:   result.Message,
			})
		}
	} else {
		// For permanent error changes the condition
		for i := range l.conditions {
			condition := &l.conditions[i]
			if condition.Type == result.Rule.Condition {
				// For permanent error only generate condition when exit status is above warning
				if result.ExitStatus >= cpmtypes.NonOK {
					// Update transition timestamp and message when the condition
					// changes. Condition is considered to be changed only when
					// status or reason changes.
					if !condition.Status || condition.Reason != result.Rule.Reason {
						condition.Transition = timestamp
						condition.Message = result.Message
					}
					condition.Status = true
					condition.Reason = result.Rule.Reason
					break
				}
			}
		}
	}
	return &types.Status{
		Source: l.config.Source,
		// TODO(random-liu): Aggregate events and conditions and then do periodically report.
		Events:     events,
		Conditions: l.conditions,
	}
}

// initializeStatus initializes the internal condition and also reports it to the node problem detector.
func (l *customPluginMonitor) initializeStatus() {
	// Initialize the default node conditions
	l.conditions = initialConditions(l.config.DefaultConditions)
	glog.Infof("Initialize condition generated: %+v", l.conditions)
	// Update the initial status
	l.statusChan <- &types.Status{
		Source:     l.config.Source,
		Conditions: l.conditions,
	}
}

func initialConditions(defaults []types.Condition) []types.Condition {
	conditions := make([]types.Condition, len(defaults))
	copy(conditions, defaults)
	for i := range conditions {
		// TODO(random-liu): Validate default conditions
		conditions[i].Status = false
		conditions[i].Transition = time.Now()
	}
	return conditions
}
