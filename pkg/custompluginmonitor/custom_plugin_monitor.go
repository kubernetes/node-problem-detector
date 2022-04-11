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
	"k8s.io/node-problem-detector/pkg/util"
	"time"

	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/custompluginmonitor/plugin"
	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemmetrics"
	"k8s.io/node-problem-detector/pkg/types"
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

	// runRules done for the interval
	intervalEndChan := c.plugin.GetIntervalEndChan()
	var intervalResults []cpmtypes.Result

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				glog.Errorf("Result channel closed: %s", c.configPath)
				return
			}

			glog.V(3).Infof("Receive new plugin result for %s: %+v", c.configPath, result)

			// gather results for single rule interval loop
			intervalResults = append(intervalResults, result)
		case _, ok := <-intervalEndChan:
			if !ok {
				glog.Errorf("Interval End Channel closed: %s", c.configPath)
				return
			}

			glog.V(3).Infof("All plugins ran for one interval for %s", c.configPath)
			status := c.generateStatus(intervalResults)
			glog.V(3).Infof("New status generated: %+v", status)
			c.statusChan <- status

			glog.V(3).Info("Resetting interval")
			intervalResults = []cpmtypes.Result{}
		case <-c.tomb.Stopping():
			c.plugin.Stop()
			glog.Infof("Custom plugin monitor stopped: %s", c.configPath)
			c.tomb.Done()
			return
		}
	}
}

func (c *customPluginMonitor) generateStatus(results []cpmtypes.Result) *types.Status {
	timestamp := time.Now()
	var activeProblemEvents []types.Event
	var inactiveProblemEvents []types.Event

	var unProcessedResults []cpmtypes.Result

	for _, result := range results {
		status := toConditionStatus(result.ExitStatus)
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
			// we skip result that sets condition true, and result that sets condition false/unknown but with a different reason
			// result that sets condition true will be processed later again
			if status == types.True {
				unProcessedResults = append(unProcessedResults, result)
				continue
			}

			for i := range c.conditions {
				condition := &c.conditions[i]

				// if appropriate (current condition's reason changes to false/unknown), unset(set to false/unknown) condition first.
				// In case there are multiple reasons per condition, this will prevent ignoring new reason that sets
				// condition true (since original condition reason takes precedence) or flapping (condition set to false
				// by current reason, then to true by another reason)
				if condition.Type == result.Rule.Condition && condition.Reason == result.Rule.Reason {

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

					var newReason string
					var newMessage string
					newReason = defaultConditionReason
					if status == types.False {
						newMessage = defaultConditionMessage
					} else {
						// When status unknown, the result's message is important for debug
						newMessage = result.Message
					}

					condition.Transition = timestamp
					condition.Status = status
					condition.Reason = newReason
					condition.Message = newMessage

					break
				}
			}
		}
	}

	for _, result := range unProcessedResults {
		status := toConditionStatus(result.ExitStatus)
		// we iterate through results that sets condition true for different reasons
		// whatever result that went through result channel first takes precedence
		for i := range c.conditions {
			condition := &c.conditions[i]
			if condition.Type == result.Rule.Condition {
				if condition.Status != types.True ||
					(condition.Reason == result.Rule.Reason && *c.config.PluginGlobalConfig.EnableMessageChangeBasedConditionUpdate) {
					// update condition only when condition is currently false/unknown, or message based condition update is enabled.
					// for each condition, this if-block will be reached once
					condition.Transition = timestamp
					condition.Status = status
					condition.Reason = result.Rule.Reason
					condition.Message = result.Message
				}

				break
			}
		}
	}

	for i := range c.conditions {
		// check for conditions that are still false/unknown
		condition := &c.conditions[i]
		updateEvent := util.GenerateConditionChangeEvent(
			condition.Type,
			condition.Status,
			condition.Reason,
			condition.Message,
			timestamp,
		)
		if condition.Status != types.True {
			inactiveProblemEvents = append(inactiveProblemEvents, updateEvent)
		} else {
			activeProblemEvents = append(activeProblemEvents, updateEvent)
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
