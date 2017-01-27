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

package kernelmonitor

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"time"

	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/util"
	"k8s.io/node-problem-detector/pkg/types"

	"github.com/golang/glog"
)

// MonitorConfig is the configuration of kernel monitor.
type MonitorConfig struct {
	// WatcherConfig is the configuration of kernel log watcher.
	WatcherConfig
	// BufferSize is the size (in lines) of the log buffer.
	BufferSize int `json:"bufferSize"`
	// Source is the source name of the kernel monitor
	Source string `json:"source"`
	// DefaultConditions are the default states of all the conditions kernel monitor should handle.
	DefaultConditions []types.Condition `json:"conditions"`
	// Rules are the rules kernel monitor will follow to parse the log file.
	Rules []kerntypes.Rule `json:"rules"`
}

// KernelMonitor monitors the kernel log and reports node problem condition and event according to
// the rules.
type KernelMonitor interface {
	// Start starts the kernel monitor.
	Start() (<-chan *types.Status, error)
	// Stop stops the kernel monitor.
	Stop()
}

type kernelMonitor struct {
	watcher    KernelLogWatcher
	buffer     LogBuffer
	config     MonitorConfig
	conditions []types.Condition
	logCh      <-chan *kerntypes.KernelLog
	output     chan *types.Status
	tomb       *util.Tomb
}

// NewKernelMonitorOrDie create a new KernelMonitor, panic if error occurs.
func NewKernelMonitorOrDie(configPath string) KernelMonitor {
	k := &kernelMonitor{
		tomb: util.NewTomb(),
	}
	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(f, &k.config)
	if err != nil {
		panic(err)
	}
	err = validateRules(k.config.Rules)
	if err != nil {
		panic(err)
	}
	glog.Infof("Finish parsing log file: %+v", k.config)
	k.watcher = NewKernelLogWatcher(k.config.WatcherConfig)
	k.buffer = NewLogBuffer(k.config.BufferSize)
	// A 1000 size channel should be big enough.
	k.output = make(chan *types.Status, 1000)
	return k
}

func (k *kernelMonitor) Start() (<-chan *types.Status, error) {
	glog.Info("Start kernel monitor")
	var err error
	k.logCh, err = k.watcher.Watch()
	if err != nil {
		return nil, err
	}
	go k.monitorLoop()
	return k.output, nil
}

func (k *kernelMonitor) Stop() {
	glog.Info("Stop kernel monitor")
	k.tomb.Stop()
}

// monitorLoop is the main loop of kernel monitor.
func (k *kernelMonitor) monitorLoop() {
	defer k.tomb.Done()
	k.initializeStatus()
	for {
		select {
		case log := <-k.logCh:
			k.parseLog(log)
		case <-k.tomb.Stopping():
			k.watcher.Stop()
			glog.Infof("Kernel monitor stopped")
			return
		}
	}
}

// parseLog parses one log line.
func (k *kernelMonitor) parseLog(log *kerntypes.KernelLog) {
	// Once there is new log, kernel monitor will push it into the log buffer and try
	// to match each rule. If any rule is matched, kernel monitor will report a status.
	k.buffer.Push(log)
	if matched := k.buffer.Match(k.config.StartPattern); len(matched) != 0 {
		// Reset the condition if a start log shows up.
		glog.Infof("Found start log %q, re-initialize the status", generateMessage(matched))
		k.initializeStatus()
		return
	}
	for _, rule := range k.config.Rules {
		matched := k.buffer.Match(rule.Pattern)
		if len(matched) == 0 {
			continue
		}
		status := k.generateStatus(matched, rule)
		glog.Infof("New status generated: %+v", status)
		k.output <- status
	}
}

// generateStatus generates status from the logs.
func (k *kernelMonitor) generateStatus(logs []*kerntypes.KernelLog, rule kerntypes.Rule) *types.Status {
	// We use the timestamp of the first log line as the timestamp of the status.
	timestamp := logs[0].Timestamp
	message := generateMessage(logs)
	var events []types.Event
	if rule.Type == kerntypes.Temp {
		// For temporary error only generate event
		events = append(events, types.Event{
			Severity:  types.Warn,
			Timestamp: timestamp,
			Reason:    rule.Reason,
			Message:   message,
		})
	} else {
		// For permanent error changes the condition
		for i := range k.conditions {
			condition := &k.conditions[i]
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
		Source: k.config.Source,
		// TODO(random-liu): Aggregate events and conditions and then do periodically report.
		Events:     events,
		Conditions: k.conditions,
	}
}

// initializeStatus initializes the internal condition and also reports it to the node problem detector.
func (k *kernelMonitor) initializeStatus() {
	// Initialize the default node conditions
	k.conditions = initialConditions(k.config.DefaultConditions)
	glog.Infof("Initialize condition generated: %+v", k.conditions)
	// Update the initial status
	k.output <- &types.Status{
		Source:     k.config.Source,
		Conditions: k.conditions,
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
func validateRules(rules []kerntypes.Rule) error {
	for _, rule := range rules {
		_, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateMessage(logs []*kerntypes.KernelLog) string {
	messages := []string{}
	for _, log := range logs {
		messages = append(messages, log.Message)
	}
	return concatLogs(messages)
}
