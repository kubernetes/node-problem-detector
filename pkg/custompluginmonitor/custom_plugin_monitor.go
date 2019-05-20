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

	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/custompluginmonitor/plugin"
	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

const CustomPluginMonitorName = "custom-plugin-monitor"

func init() {
	problemdaemon.Register(
		CustomPluginMonitorName,
		types.ProblemDaemonHandler{
			CreateProblemDaemonOrDie: NewCustomPluginMonitorOrDie,
			CmdOptionDescription:     "Set to config file paths."})
}

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
	c := &customPluginMonitor{
		tomb: tomb.NewTomb(),
	}
	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		glog.Fatalf("Failed to read configuration file %q: %v", configPath, err)
	}
	err = json.Unmarshal(f, &c.config)
	if err != nil {
		glog.Fatalf("Failed to unmarshal configuration file %q: %v", configPath, err)
	}
	// Apply configurations
	err = (&c.config).ApplyConfiguration()
	if err != nil {
		glog.Fatalf("Failed to apply configuration for %q: %v", configPath, err)
	}

	// Validate configurations
	err = c.config.Validate()
	if err != nil {
		glog.Fatalf("Failed to validate custom plugin config %+v: %v", c.config, err)
	}

	glog.Infof("Finish parsing custom plugin monitor config file: %+v", c.config)

	c.plugin = plugin.NewPlugin(c.config)
	// A 1000 size channel should be big enough.
	c.statusChan = make(chan *types.Status, 1000)
	return c
}

func (c *customPluginMonitor) Start() (<-chan *types.Status, error) {
	glog.Info("Start custom plugin monitor")
	go c.plugin.Run()
	go c.monitorLoop()
	return c.statusChan, nil
}

func (c *customPluginMonitor) Stop() {
	glog.Info("Stop custom plugin monitor")
	c.tomb.Stop()
}

// monitorLoop is the main loop of log monitor.
func (c *customPluginMonitor) monitorLoop() {
	c.initializeStatus()

	resultChan := c.plugin.GetResultChan()

	for {
		select {
		case result := <-resultChan:
			glog.V(3).Infof("Receive new plugin result: %+v", result)
			status := c.generateStatus(result)
			glog.Infof("New status generated: %+v", status)
			c.statusChan <- status
		case <-c.tomb.Stopping():
			c.plugin.Stop()
			glog.Infof("Custom plugin monitor stopped")
			c.tomb.Done()
			break
		}
	}
}

// generateStatus generates status from the plugin check result.
func (c *customPluginMonitor) generateStatus(result cpmtypes.Result) *types.Status {
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
		for i := range c.conditions {
			condition := &c.conditions[i]
			if condition.Type == result.Rule.Condition {
				status := toConditionStatus(result.ExitStatus)
				// change 1: Condition status change from True to False/Unknown
				if condition.Status == types.True && status != types.True {
					condition.Transition = timestamp
					var defaultConditionReason string
					var defaultConditionMessage string
					for j := range c.config.DefaultConditions {
						defaultCondition := &c.config.DefaultConditions[j]
						if defaultCondition.Type == result.Rule.Condition {
							defaultConditionReason = defaultCondition.Reason
							defaultConditionMessage = defaultCondition.Message
							break
						}
					}

					events = append(events, util.GenerateConditionChangeEvent(
						condition.Type,
						status,
						defaultConditionReason,
						timestamp,
					))

					condition.Status = status
					condition.Message = defaultConditionMessage
					condition.Reason = defaultConditionReason
				} else if condition.Status != types.True && status == types.True {
					// change 2: Condition status change from False/Unknown to True
					condition.Transition = timestamp
					condition.Message = result.Message
					events = append(events, util.GenerateConditionChangeEvent(
						condition.Type,
						status,
						result.Rule.Reason,
						timestamp,
					))

					condition.Status = status
					condition.Reason = result.Rule.Reason
				} else if condition.Status != status {
					// change 3: Condition status change from False to Unknown or vice versa
					condition.Transition = timestamp
					condition.Message = result.Message
					events = append(events, util.GenerateConditionChangeEvent(
						condition.Type,
						status,
						result.Rule.Reason,
						timestamp,
					))

					condition.Status = status
					condition.Reason = result.Rule.Reason
				} else if condition.Status == status &&
					(condition.Reason != result.Rule.Reason ||
						(*c.config.PluginGlobalConfig.EnableMessageChangeBasedConditionUpdate && condition.Message != result.Message)) {
					// change 4: Condition status do not change.
					// condition reason changes or
					// condition message changes when message based condition update is enabled.
					condition.Transition = timestamp
					condition.Reason = result.Rule.Reason
					condition.Message = result.Message
					events = append(events, util.GenerateConditionChangeEvent(
						condition.Type,
						status,
						condition.Reason,
						timestamp,
					))
				}

				break
			}
		}
	}
	return &types.Status{
		Source: c.config.Source,
		// TODO(random-liu): Aggregate events and conditions and then do periodically report.
		Events:     events,
		Conditions: c.conditions,
	}
}

func toConditionStatus(s cpmtypes.Status) types.ConditionStatus {
	switch s {
	case cpmtypes.OK:
		return types.False
	case cpmtypes.NonOK:
		return types.True
	default:
		return types.Unknown
	}
}

// initializeStatus initializes the internal condition and also reports it to the node problem detector.
func (c *customPluginMonitor) initializeStatus() {
	// Initialize the default node conditions
	c.conditions = initialConditions(c.config.DefaultConditions)
	glog.Infof("Initialize condition generated: %+v", c.conditions)
	// Update the initial status
	c.statusChan <- &types.Status{
		Source:     c.config.Source,
		Conditions: c.conditions,
	}
}

func initialConditions(defaults []types.Condition) []types.Condition {
	conditions := make([]types.Condition, len(defaults))
	copy(conditions, defaults)
	for i := range conditions {
		conditions[i].Status = types.False
		conditions[i].Transition = time.Now()
	}
	return conditions
}
