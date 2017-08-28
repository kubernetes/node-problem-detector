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

package systemlogmonitor

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"time"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers"
	watchertypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"
	logtypes "k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
	"k8s.io/node-problem-detector/pkg/systemlogmonitor/util"
	"k8s.io/node-problem-detector/pkg/types"

	"github.com/golang/glog"
)

// LogMonitor monitors the log and reports node problem condition and event according to
// the rules.
type LogMonitor interface {
	// Start starts the log monitor.
	Start() (<-chan *types.Status, error)
	// Stop stops the log monitor.
	Stop()
}

type logMonitor struct {
	watcher    watchertypes.LogWatcher
	buffer     LogBuffer
	config     MonitorConfig
	conditions []types.Condition
	logCh      <-chan *logtypes.Log
	output     chan *types.Status
	tomb       *util.Tomb
}

// NewLogMonitorOrDie create a new LogMonitor, panic if error occurs.
func NewLogMonitorOrDie(configPath string) LogMonitor {
	l := &logMonitor{
		tomb: util.NewTomb(),
	}
	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		glog.Fatalf("Failed to read configuration file %q: %v", configPath, err)
	}
	err = json.Unmarshal(f, &l.config)
	if err != nil {
		glog.Fatalf("Failed to unmarshal configuration file %q: %v", configPath, err)
	}
	// Apply default configurations
	applyDefaultConfiguration(&l.config)
	err = validateRules(l.config.Rules)
	if err != nil {
		glog.Fatalf("Failed to validate matching rules %#v: %v", l.config.Rules, err)
	}
	glog.Infof("Finish parsing log monitor config file: %+v", l.config)
	l.watcher = logwatchers.GetLogWatcherOrDie(l.config.WatcherConfig)
	l.buffer = NewLogBuffer(l.config.BufferSize)
	// A 1000 size channel should be big enough.
	l.output = make(chan *types.Status, 1000)

	return l
}

func (l *logMonitor) Start() (<-chan *types.Status, error) {
	glog.Info("Start log monitor")
	var err error
	l.logCh, err = l.watcher.Watch()
	if err != nil {
		return nil, err
	}
	go l.monitorLoop()
	return l.output, nil
}

func (l *logMonitor) Stop() {
	glog.Info("Stop log monitor")
	l.tomb.Stop()
}

// monitorLoop is the main loop of log monitor.
func (l *logMonitor) monitorLoop() {
	defer l.tomb.Done()
	l.initializeStatus()
	for {
		select {
		case log := <-l.logCh:
			l.parseLog(log)
		case <-l.tomb.Stopping():
			l.watcher.Stop()
			glog.Infof("Log monitor stopped")
			return
		}
	}
}

// parseLog parses one log line.
func (l *logMonitor) parseLog(log *logtypes.Log) {
	// Once there is new log, log monitor will push it into the log buffer and try
	// to match each rule. If any rule is matched, log monitor will report a status.
	l.buffer.Push(log)
	for _, rule := range l.config.Rules {
		matched := l.buffer.Match(rule.Pattern)
		if len(matched) == 0 {
			continue
		}
		status := l.generateStatus(matched, rule)
		glog.Infof("New status generated: %+v", status)
		l.output <- status
	}
}

// generateStatus generates status from the logs.
func (l *logMonitor) generateStatus(logs []*logtypes.Log, rule logtypes.Rule) *types.Status {
	// We use the timestamp of the first log line as the timestamp of the status.
	timestamp := logs[0].Timestamp
	message := generateMessage(logs)
	var events []types.Event
	if rule.Type == logtypes.Temp {
		// For temporary error only generate event
		events = append(events, types.Event{
			Severity:  types.Warn,
			Timestamp: timestamp,
			Reason:    rule.Reason,
			Message:   message,
		})
	} else {
		// For permanent error changes the condition
		for i := range l.conditions {
			condition := &l.conditions[i]
			if condition.Type == rule.Condition {
				condition.Type = rule.Condition
				// Update transition timestamp and message when the condition
				// changes. Condition is considered to be changed only when
				// status or reason changes.
				if !condition.Status || condition.Reason != rule.Reason {
					condition.Transition = timestamp
					condition.Message = message
				}
				condition.Status = true
				condition.Reason = rule.Reason
				break
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
func (l *logMonitor) initializeStatus() {
	// Initialize the default node conditions
	l.conditions = initialConditions(l.config.DefaultConditions)
	glog.Infof("Initialize condition generated: %+v", l.conditions)
	// Update the initial status
	l.output <- &types.Status{
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

// validateRules verifies whether the regular expressions in the rules are valid.
func validateRules(rules []logtypes.Rule) error {
	for _, rule := range rules {
		_, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateMessage(logs []*logtypes.Log) string {
	messages := []string{}
	for _, log := range logs {
		messages = append(messages, log.Message)
	}
	return concatLogs(messages)
}
