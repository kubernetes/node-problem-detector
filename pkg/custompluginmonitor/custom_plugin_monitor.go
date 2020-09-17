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
	"k8s.io/node-problem-detector/pkg/problemmetrics"
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
	configPath string
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
		configPath: configPath,
		tomb:       tomb.NewTomb(),
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

	glog.Infof("Finish parsing custom plugin monitor config file %s: %+v", c.configPath, c.config)

	c.plugin = plugin.NewPlugin(c.config)
	// A 1000 size channel should be big enough.
	c.statusChan = make(chan *types.Status, 1000)

	if *c.config.EnableMetricsReporting {
		initializeProblemMetricsOrDie(c.config.Rules)
	}
	return c
}

// initializeProblemMetricsOrDie creates problem metrics for all problems and set the value to 0,
// panic if error occurs.
func initializeProblemMetricsOrDie(rules []*cpmtypes.CustomRule) {
	for _, rule := range rules {
		if rule.Type == types.Perm {
			err := problemmetrics.GlobalProblemMetricsManager.SetProblemGauge(rule.Condition, rule.Reason, false)
			if err != nil {
				glog.Fatalf("Failed to initialize problem gauge metrics for problem %q, reason %q: %v",
					rule.Condition, rule.Reason, err)
			}
		}
		err := problemmetrics.GlobalProblemMetricsManager.IncrementProblemCounter(rule.Reason, 0)
		if err != nil {
			glog.Fatalf("Failed to initialize problem counter metrics for %q: %v", rule.Reason, err)
		}
	}
}

func (c *customPluginMonitor) Start() (<-chan *types.Status, error) {
	glog.Infof("Start custom plugin monitor %s", c.configPath)
	go c.plugin.Run()
	go c.monitorLoop()
	return c.statusChan, nil
}

func (c *customPluginMonitor) Stop() {
	glog.Infof("Stop custom plugin monitor %s", c.configPath)
	c.tomb.Stop()
}

// monitorLoop is the main loop of customPluginMonitor.
func (c *customPluginMonitor) monitorLoop() {
	c.initializeStatus()

	resultChan := c.plugin.GetResultChan()

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				glog.Errorf("Result channel closed: %s", c.configPath)
				return
			}
			glog.V(3).Infof("Receive new plugin result for %s: %+v", c.configPath, result)
			status := c.generateStatus(result)
			glog.V(3).Infof("New status generated: %+v", status)
			c.statusChan <- status
		case <-c.tomb.Stopping():
			c.plugin.Stop()
			glog.Infof("Custom plugin monitor stopped: %s", c.configPath)
			c.tomb.Done()
			return
		}
	}
}

// generateStatus generates status from the plugin check result.
func (c *customPluginMonitor) generateStatus(result cpmtypes.Result) *types.Status {
	timestamp := time.Now()
	var activeProblemEvents []types.Event
	var inactiveProblemEvents []types.Event
	if result.Rule.Type == types.Temp {
		// For temporary error only generate event when exit status is above warning
		if result.ExitStatus >= cpmtypes.NonOK {
			activeProblemEvents = append(activeProblemEvents, types.Event{
				Severity:  types.Warn,
				Timestamp: timestamp,
				Reason:    result.Rule.Reason,
				Message:   result.Message,
			})
		}
	} else {
		// For permanent error that changes the condition
		for i := range c.conditions {
			condition := &c.conditions[i]
			if condition.Type == result.Rule.Condition {
				// The condition reason specified in the rule and the result message
				// represent the problem happened. We need to know the default condition
				// from the config, so that we can set the new condition reason/message
				// back when such problem goes away.
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

				needToUpdateCondition := true
				var newReason string
				var newMessage string
				status := toConditionStatus(result.ExitStatus)
				if condition.Status == types.True && status != types.True {
					// Scenario 1: Condition status changes from True to False/Unknown
					newReason = defaultConditionReason
					if newMessage == "" {
						newMessage = defaultConditionMessage
					} else {
						newMessage = result.Message
					}
				} else if condition.Status != types.True && status == types.True {
					// Scenario 2: Condition status changes from False/Unknown to True
					newReason = result.Rule.Reason
					newMessage = result.Message
				} else if condition.Status != status {
					// Scenario 3: Condition status changes from False to Unknown or vice versa
					newReason = defaultConditionReason
					if newMessage == "" {
						newMessage = defaultConditionMessage
					} else {
						newMessage = result.Message
					}
				} else if condition.Status == types.True && status == types.True &&
					(condition.Reason != result.Rule.Reason ||
						(*c.config.PluginGlobalConfig.EnableMessageChangeBasedConditionUpdate && condition.Message != result.Message)) {
					// Scenario 4: Condition status does not change and it stays true.
					// condition reason changes or
					// condition message changes when message based condition update is enabled.
					newReason = result.Rule.Reason
					newMessage = result.Message
				} else {
					// Scenario 5: Condition status does not change and it stays False/Unknown.
					// This should just be the default reason or message (as a consequence
					// of scenario 1 and scenario 3 above).
					needToUpdateCondition = false
				}

				if needToUpdateCondition {
					condition.Transition = timestamp
					condition.Status = status
					condition.Reason = newReason
					condition.Message = newMessage

					updateEvent := util.GenerateConditionChangeEvent(
						condition.Type,
						status,
						newReason,
						timestamp,
					)

					if status == types.True {
						activeProblemEvents = append(activeProblemEvents, updateEvent)
					} else {
						inactiveProblemEvents = append(inactiveProblemEvents, updateEvent)
					}
				}

				break
			}
		}
	}
	if *c.config.EnableMetricsReporting {
		// Increment problem counter only for active problems which just got detected.
		for _, event := range activeProblemEvents {
			err := problemmetrics.GlobalProblemMetricsManager.IncrementProblemCounter(
				event.Reason, 1)
			if err != nil {
				glog.Errorf("Failed to update problem counter metrics for %q: %v",
					event.Reason, err)
			}
		}
		for _, condition := range c.conditions {
			err := problemmetrics.GlobalProblemMetricsManager.SetProblemGauge(
				condition.Type, condition.Reason, condition.Status == types.True)
			if err != nil {
				glog.Errorf("Failed to update problem gauge metrics for problem %q, reason %q: %v",
					condition.Type, condition.Reason, err)
			}
		}
	}
	status := &types.Status{
		Source: c.config.Source,
		// TODO(random-liu): Aggregate events and conditions and then do periodically report.
		Events:     append(activeProblemEvents, inactiveProblemEvents...),
		Conditions: c.conditions,
	}
	// Log only if condition has changed
	if len(activeProblemEvents) != 0 || len(inactiveProblemEvents) != 0 {
		glog.V(0).Infof("New status generated: %+v", status)
	}
	return status
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
