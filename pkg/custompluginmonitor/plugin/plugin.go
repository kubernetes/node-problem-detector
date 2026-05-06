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

package plugin

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"k8s.io/klog/v2"

	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

// maxCustomPluginBufferBytes is the max bytes that a custom plugin is allowed to
// send to stdout/stderr. Any bytes exceeding this value will be truncated.
const maxCustomPluginBufferBytes = 1024 * 4

type Plugin struct {
	config     cpmtypes.CustomPluginConfig
	syncChan   chan struct{}
	resultChan chan cpmtypes.Result
	tomb       *tomb.Tomb
	sync.WaitGroup
}

func NewPlugin(config cpmtypes.CustomPluginConfig) *Plugin {
	return &Plugin{
		config:   config,
		syncChan: make(chan struct{}, *config.PluginGlobalConfig.Concurrency),
		// A 1000 size channel should be big enough.
		resultChan: make(chan cpmtypes.Result, 1000),
		tomb:       tomb.NewTomb(),
	}
}

func (p *Plugin) GetResultChan() <-chan cpmtypes.Result {
	return p.resultChan
}

func (p *Plugin) Run() {
	defer func() {
		klog.Info("Stopping plugin execution")
		close(p.resultChan)
		p.tomb.Done()
	}()

	runTicker := time.NewTicker(*p.config.PluginGlobalConfig.InvokeInterval)
	defer runTicker.Stop()

	// on boot run once
	select {
	case <-p.tomb.Stopping():
		return
	default:
		p.runRules()
	}

	// run every InvokeInterval
	for {
		select {
		case <-runTicker.C:
			p.runRules()
		case <-p.tomb.Stopping():
			return
		}
	}
}

// run each rule in parallel and wait for them to complete
func (p *Plugin) runRules() {
	klog.V(3).Info("Start to run custom plugins")

	for _, rule := range p.config.Rules {
		// syncChan limits concurrent goroutines to configured PluginGlobalConfig.Concurrency value
		p.syncChan <- struct{}{}
		p.Add(1)
		go func(rule *cpmtypes.CustomRule) {
			defer p.Done()
			defer func() {
				<-p.syncChan
			}()

			start := time.Now()
			exitStatus, message := p.run(*rule)
			level := klog.Level(3)
			if exitStatus != 0 {
				level = klog.Level(2)
			}

			klog.V(level).Infof("Rule: %+v. Start time: %v. End time: %v. Duration: %v", rule, start, time.Now(), time.Since(start))

			result := cpmtypes.Result{
				Rule:       rule,
				ExitStatus: exitStatus,
				Message:    message,
			}

			// pipes result into resultChan which customPluginMonitor instance generates status from
			p.resultChan <- result

			// Let the result be logged at a higher verbosity level. If there is a change in status it is logged later.
			klog.V(level).Infof("Add check result %+v for rule %+v", result, rule)
		}(rule)
	}

	p.Wait()
	klog.V(3).Info("Finish running custom plugins")
}

// readFromReader reads the maxBytes from the reader and drains the rest.
func readFromReader(reader io.ReadCloser, maxBytes int64) ([]byte, error) {
	limitReader := io.LimitReader(reader, maxBytes)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return []byte{}, err
	}
	// Drain the reader
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (p *Plugin) run(rule cpmtypes.CustomRule) (exitStatus cpmtypes.Status, output string) {
	isTimeout := false
	isHung := false

	var timeoutDuration time.Duration
	if rule.Timeout != nil && *rule.Timeout < *p.config.PluginGlobalConfig.Timeout {
		timeoutDuration = *rule.Timeout
	} else {
		timeoutDuration = *p.config.PluginGlobalConfig.Timeout
	}

	cmd := util.Exec(ctx, rule.Path, rule.Args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		klog.Errorf("Error creating stdout pipe for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error creating stdout pipe for plugin. Please check the error log"
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		klog.Errorf("Error creating stderr pipe for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error creating stderr pipe for plugin. Please check the error log"
	}
	if err := cmd.Start(); err != nil {
		klog.Errorf("Error in starting plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error in starting plugin. Please check the error log"
	}

	var (
		wg        sync.WaitGroup
		stdout    []byte
		stderr    []byte
		stdoutErr error
		stderrErr error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		stdout, stdoutErr = readFromReader(stdoutPipe, maxCustomPluginBufferBytes)
	}()
	go func() {
		defer wg.Done()
		stderr, stderrErr = readFromReader(stderrPipe, maxCustomPluginBufferBytes)
	}()
	// This will wait for the reads to complete. If the execution times out, the pipes
	// will be closed and the wait group unblocks.
	// If the timeout is caused by the plugin process or sub-process hung due to GPU device errors or other reasons,
	// wg.Wait() will be blocked forever, so we need to add a timeout to the wait group.
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()
	select {
	case <-waitChan:
		// The reads are done.
		break
	case <-time.After(timeoutDuration):
		klog.Errorf("Waiting for command output timed out when running plugin %q", rule.Path)
		isTimeout = true
		err := util.Kill(cmd)
		if err != nil {
			klog.Errorf("Error when killing process %d: %v", cmd.Process.Pid, err)
		} else {
			klog.Infof("Killed process %d successfully", cmd.Process.Pid)
		}

		// Check if the process is in D state. If it is, the process is hung and can not be killed.
		// It also means that the plugin can not report the correct status, instead reports Unknown status.
		// On a GPU machine, a plugin with Python script calling pynvml API may hang in D state due to some GPU device errors.
		if util.IsProcessInDState(cmd.Process.Pid) {
			klog.Errorf("Process %d is hung in D state", cmd.Process.Pid)
			isHung = true
		}
	}

	if isHung {
		return cpmtypes.Unknown, fmt.Sprintf("Process is hung when running plugin %s", rule.Path)
	}

	if !isTimeout && stdoutErr != nil {
		klog.Errorf("Error reading stdout for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error reading stdout for plugin. Please check the error log"
	}

	if !isTimeout && stderrErr != nil {
		klog.Errorf("Error reading stderr for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error reading stderr for plugin. Please check the error log"
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			klog.Errorf("Error in waiting for plugin %q: error - %v. output - %q", rule.Path, err, string(stdout))
			return cpmtypes.Unknown, "Error in waiting for plugin. Please check the error log"
		}
	}

	stderrStr := ""
	if isTimeout {
		output = fmt.Sprintf("Timeout when running plugin %q: state - %s. output - %q", rule.Path, cmd.ProcessState.String(), "")
	} else {
		// trim suffix useless bytes
		output = strings.TrimSpace(string(stdout))
		stderrStr = strings.TrimSpace(string(stderr))
	}

	// cut at position max_output_length if stdout is longer than max_output_length bytes
	if len(output) > *p.config.PluginGlobalConfig.MaxOutputLength {
		output = output[:*p.config.PluginGlobalConfig.MaxOutputLength]
	}

	exitCode := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
	switch exitCode {
	case 0:
		logPluginStderr(rule, stderrStr, 3)
		return cpmtypes.OK, output
	case 1:
		logPluginStderr(rule, stderrStr, 0)
		return cpmtypes.NonOK, output
	default:
		logPluginStderr(rule, stderrStr, 0)
		return cpmtypes.Unknown, output
	}
}

// Stop the plugin.
func (p *Plugin) Stop() {
	p.tomb.Stop()
	klog.Info("Stop plugin execution")
}

func logPluginStderr(rule cpmtypes.CustomRule, logs string, logLevel klog.Level) {
	if len(logs) != 0 {
		klog.V(logLevel).Infof("Start logs from plugin %+v \n %s", rule, logs)
		klog.V(logLevel).Infof("End logs from plugin %+v", rule)
	}
}
