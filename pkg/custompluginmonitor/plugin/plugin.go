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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
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
		glog.Info("Stopping plugin execution")
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
	glog.V(3).Info("Start to run custom plugins")

	for _, rule := range p.config.Rules {
		p.syncChan <- struct{}{}
		p.Add(1)
		go func(rule *cpmtypes.CustomRule) {
			defer p.Done()
			defer func() {
				<-p.syncChan
			}()

			start := time.Now()
			exitStatus, message := p.run(*rule)

			glog.V(3).Infof("Rule: %+v. Start time: %v. End time: %v. Duration: %v", rule, start, time.Now(), time.Since(start))

			result := cpmtypes.Result{
				Rule:       rule,
				ExitStatus: exitStatus,
				Message:    message,
			}

			p.resultChan <- result

			// Let the result be logged at a higher verbosity level. If there is a change in status it is logged later.
			glog.V(3).Infof("Add check result %+v for rule %+v", result, rule)
		}(rule)
	}

	p.Wait()
	glog.V(3).Info("Finish running custom plugins")
}

// readFromReader reads the maxBytes from the reader and drains the rest.
func readFromReader(reader io.ReadCloser, maxBytes int64) ([]byte, error) {
	limitReader := io.LimitReader(reader, maxBytes)
	data, err := ioutil.ReadAll(limitReader)
	if err != nil {
		return []byte{}, err
	}
	// Drain the reader
	if _, err := io.Copy(ioutil.Discard, reader); err != nil {
		return []byte{}, err
	}
	return data, nil
}

func (p *Plugin) run(rule cpmtypes.CustomRule) (exitStatus cpmtypes.Status, output string) {
	var ctx context.Context
	var cancel context.CancelFunc

	if rule.Timeout != nil && *rule.Timeout < *p.config.PluginGlobalConfig.Timeout {
		ctx, cancel = context.WithTimeout(context.Background(), *rule.Timeout)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), *p.config.PluginGlobalConfig.Timeout)
	}
	defer cancel()

	// create a process group
	sysProcAttr := &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd := exec.Command(rule.Path, rule.Args...)
	cmd.SysProcAttr = sysProcAttr

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		glog.Errorf("Error creating stdout pipe for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error creating stdout pipe for plugin. Please check the error log"
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		glog.Errorf("Error creating stderr pipe for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error creating stderr pipe for plugin. Please check the error log"
	}
	if err := cmd.Start(); err != nil {
		glog.Errorf("Error in starting plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error in starting plugin. Please check the error log"
	}

	waitChan := make(chan struct{})
	defer close(waitChan)

	go func() {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return
			}
			glog.Errorf("Error in running plugin timeout %q", rule.Path)
			if cmd.Process == nil || cmd.Process.Pid == 0 {
				glog.Errorf("Error in cmd.Process check %q", rule.Path)
				break
			}
			err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			if err != nil {
				glog.Errorf("Error in kill process %d, %v", cmd.Process.Pid, err)
			}
		case <-waitChan:
			return
		}
	}()

	var (
		wg        sync.WaitGroup
		stdout    []byte
		stderr    []byte
		stdoutErr error
		stderrErr error
	)

	wg.Add(2)
	go func() {
		stdout, stdoutErr = readFromReader(stdoutPipe, maxCustomPluginBufferBytes)
		wg.Done()
	}()
	go func() {
		stderr, stderrErr = readFromReader(stderrPipe, maxCustomPluginBufferBytes)
		wg.Done()
	}()
	// This will wait for the reads to complete. If the execution times out, the pipes
	// will be closed and the wait group unblocks.
	wg.Wait()

	if stdoutErr != nil {
		glog.Errorf("Error reading stdout for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error reading stdout for plugin. Please check the error log"
	}

	if stderrErr != nil {
		glog.Errorf("Error reading stderr for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error reading stderr for plugin. Please check the error log"
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			glog.Errorf("Error in waiting for plugin %q: error - %v. output - %q", rule.Path, err, string(stdout))
			return cpmtypes.Unknown, "Error in waiting for plugin. Please check the error log"
		}
	}

	// trim suffix useless bytes
	output = string(stdout)
	output = strings.TrimSpace(output)

	if cmd.ProcessState.Sys().(syscall.WaitStatus).Signaled() {
		output = fmt.Sprintf("Timeout when running plugin %q: state - %s. output - %q", rule.Path, cmd.ProcessState.String(), output)
	}

	// cut at position max_output_length if stdout is longer than max_output_length bytes
	if len(output) > *p.config.PluginGlobalConfig.MaxOutputLength {
		output = output[:*p.config.PluginGlobalConfig.MaxOutputLength]
	}

	exitCode := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
	switch exitCode {
	case 0:
		logPluginStderr(rule, string(stderr), 3)
		return cpmtypes.OK, output
	case 1:
		logPluginStderr(rule, string(stderr), 0)
		return cpmtypes.NonOK, output
	default:
		logPluginStderr(rule, string(stderr), 0)
		return cpmtypes.Unknown, output
	}
}

func (p *Plugin) Stop() {
	p.tomb.Stop()
	glog.Info("Stop plugin execution")
}

func logPluginStderr(rule cpmtypes.CustomRule, logs string, logLevel glog.Level) {
	if len(logs) != 0 {
		glog.V(logLevel).Infof("Start logs from plugin %+v \n %s", rule, logs)
		glog.V(logLevel).Infof("End logs from plugin %+v", rule)
	}
}
