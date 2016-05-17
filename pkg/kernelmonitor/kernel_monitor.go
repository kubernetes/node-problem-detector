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
	"syscall"
	"time"

	kerntypes "k8s.io/node-problem-detector/pkg/kernelmonitor/types"
	"k8s.io/node-problem-detector/pkg/kernelmonitor/util"
	"k8s.io/node-problem-detector/pkg/types"

	"github.com/golang/glog"
)

// May want to add more conditions if we need finer grained node conditions.
// TODO(random-liu): Make the kernel condition to be a predefined list, and make it configurable
// in rule.
const (
	KernelDeadlockCondition = "KernelDeadlock"
	KernelMonitorSource     = "kernel-monitor"
)

// MonitorConfig is the configuration of kernel monitor.
type MonitorConfig struct {
	// WatcherConfig is the configuration of kernel log watcher.
	WatcherConfig
	// BufferSize is the size (in lines) of the log buffer.
	BufferSize int `json:"bufferSize"`
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
	watcher   KernelLogWatcher
	buffer    LogBuffer
	config    MonitorConfig
	condition types.Condition
	uptime    time.Time
	logCh     <-chan *kerntypes.KernelLog
	output    chan *types.Status
	tomb      *util.Tomb
}

// NewKernelMonitorOrDie create a new KernelMonitor, panic if error occurs.
func NewKernelMonitorOrDie(configPath string) KernelMonitor {
	k := &kernelMonitor{
		condition: defaultCondition(),
		tomb:      util.NewTomb(),
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
	var info syscall.Sysinfo_t
	err = syscall.Sysinfo(&info)
	if err != nil {
		panic(err)
	}
	k.uptime = time.Now().Add(time.Duration(-info.Uptime * int64(time.Second)))
	glog.Infof("Got system boot time: %v", k.uptime)
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
	k.output <- defaultStatus() // Update the default status
	for {
		select {
		case log := <-k.logCh:
			// Once there is new log, kernel monitor will push it into the log buffer and try
			// to match each rule. If any rule is matched, kernel monitor will report a status.
			k.buffer.Push(log)
			for _, rule := range k.config.Rules {
				matched := k.buffer.Match(rule.Pattern)
				if len(matched) == 0 {
					continue
				}
				status := k.generateStatus(matched, rule)
				glog.Infof("New status generated: %+v", status)
				k.output <- status
			}
		case <-k.tomb.Stopping():
			k.watcher.Stop()
			glog.Infof("Kernel monitor stopped")
			return
		}
	}
}

// generateStatus generates status from the logs.
func (k *kernelMonitor) generateStatus(logs []*kerntypes.KernelLog, rule kerntypes.Rule) *types.Status {
	// We use the timestamp of the first log line as the timestamp of the status.
	timestamp := k.generateTimestamp(logs[0].Timestamp)
	messages := []string{}
	for _, log := range logs {
		messages = append(messages, log.Message)
	}
	message := concatLogs(messages)
	var event *types.Event
	if rule.Type == kerntypes.Temp {
		// For temporary error only generate event
		event = &types.Event{
			Severity:  types.Warn,
			Timestamp: timestamp,
			Reason:    rule.Reason,
			Message:   message,
		}
	} else {
		// For permanent error changes the condition
		k.condition.Type = KernelDeadlockCondition
		k.condition.Status = true
		k.condition.Transition = timestamp
		k.condition.Reason = rule.Reason
		k.condition.Message = message
	}
	return &types.Status{
		Source:    KernelMonitorSource,
		Event:     event,
		Condition: k.condition,
	}
}

// generateTimestamp converts the kernel log time to real time.
func (k *kernelMonitor) generateTimestamp(timestamp int64) time.Time {
	return k.uptime.Add(time.Duration(timestamp * int64(time.Microsecond)))
}

// defaultStatus returns the default status with default condition.
func defaultStatus() *types.Status {
	return &types.Status{
		Source:    KernelMonitorSource,
		Condition: defaultCondition(),
	}
}

func defaultCondition() types.Condition {
	return types.Condition{
		Type:       KernelDeadlockCondition,
		Status:     false,
		Transition: time.Now(),
	}
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
