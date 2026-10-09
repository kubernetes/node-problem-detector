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
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"k8s.io/klog/v2"
	"k8s.io/utils/clock"

	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

// maxCustomPluginStdoutCeilingBytes is a hard safety ceiling on how many bytes
// of stdout NPD reads from a custom plugin, bounding memory use from a runaway
// plugin. The per-plugin max_output_length is what actually governs the message
// size; this ceiling only caps the (misconfigured) case where max_output_length
// itself exceeds it. It must be >= the largest supported max_output_length so a
// plugin's configured limit is never silently truncated by the read buffer.
const maxCustomPluginStdoutCeilingBytes = 1024 * 1024

// maxCustomPluginStderrBytes caps stderr. run() uses stderr only for diagnostic
// logging (logPluginStderr) and never includes it in the returned output, so it
// does not need to honor max_output_length and keeps a small fixed cap.
const maxCustomPluginStderrBytes = 1024 * 4

type Plugin struct {
	config     cpmtypes.CustomPluginConfig
	syncChan   chan struct{}
	resultChan chan cpmtypes.Result
	tomb       *tomb.Tomb
	clock      clock.WithTicker
	runFunc    func(cpmtypes.CustomRule) (cpmtypes.Status, string)
	sync.WaitGroup
}

type intervalGroup struct {
	interval time.Duration
	rules    []*cpmtypes.CustomRule
	ticker   clock.Ticker
}

func NewPlugin(config cpmtypes.CustomPluginConfig) *Plugin {
	p := &Plugin{
		config:   config,
		syncChan: make(chan struct{}, *config.PluginGlobalConfig.Concurrency),
		// A 1000 size channel should be big enough.
		resultChan: make(chan cpmtypes.Result, 1000),
		tomb:       tomb.NewTomb(),
		clock:      clock.RealClock{},
	}
	p.runFunc = p.run
	return p
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

	groups := p.intervalGroups()
	if len(groups) == 0 {
		<-p.tomb.Stopping()
		return
	}

	for i := range groups {
		groups[i].ticker = p.clock.NewTicker(groups[i].interval)
	}
	defer func() {
		for i := range groups {
			groups[i].ticker.Stop()
		}
	}()

	// On boot, run every rule in one batch.
	if !p.runRules(p.config.Rules) {
		return
	}

	for i := range groups {
		p.Add(1)
		go p.runGroup(&groups[i])
	}
	p.Wait()
}

func (p *Plugin) intervalGroups() []intervalGroup {
	groups := []intervalGroup{}
	groupIndexes := make(map[time.Duration]int)
	for _, rule := range p.config.Rules {
		interval := p.effectiveInterval(rule)
		groupIndex, ok := groupIndexes[interval]
		if !ok {
			groupIndex = len(groups)
			groupIndexes[interval] = groupIndex
			groups = append(groups, intervalGroup{interval: interval})
		}
		groups[groupIndex].rules = append(groups[groupIndex].rules, rule)
	}
	return groups
}

func (p *Plugin) effectiveInterval(rule *cpmtypes.CustomRule) time.Duration {
	if rule.InvokeInterval != nil {
		return *rule.InvokeInterval
	}
	return *p.config.PluginGlobalConfig.InvokeInterval
}

func (p *Plugin) runGroup(group *intervalGroup) {
	defer p.Done()
	for {
		select {
		case <-group.ticker.C():
			if !p.runRules(group.rules) {
				return
			}
		case <-p.tomb.Stopping():
			return
		}
	}
}

// runRules runs each rule in parallel and waits for the batch to complete.
func (p *Plugin) runRules(rules []*cpmtypes.CustomRule) bool {
	klog.V(3).Info("Start to run custom plugins")
	var workers sync.WaitGroup

	for _, rule := range rules {
		// syncChan limits concurrent goroutines to configured PluginGlobalConfig.Concurrency value
		select {
		case p.syncChan <- struct{}{}:
		case <-p.tomb.Stopping():
			workers.Wait()
			return false
		}

		select {
		case <-p.tomb.Stopping():
			<-p.syncChan
			workers.Wait()
			return false
		default:
		}

		workers.Add(1)
		go func(rule *cpmtypes.CustomRule) {
			defer workers.Done()
			defer func() {
				<-p.syncChan
			}()

			select {
			case <-p.tomb.Stopping():
				return
			default:
			}

			start := time.Now()
			exitStatus, message := p.runFunc(*rule)
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
			select {
			case p.resultChan <- result:
			case <-p.tomb.Stopping():
				return
			}

			// Let the result be logged at a higher verbosity level. If there is a change in status it is logged later.
			klog.V(level).Infof("Add check result %+v for rule %+v", result, rule)
		}(rule)
	}

	workers.Wait()
	klog.V(3).Info("Finish running custom plugins")
	return true
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
	var ctx context.Context
	var cancel context.CancelFunc

	if rule.Timeout != nil && *rule.Timeout < *p.config.PluginGlobalConfig.Timeout {
		ctx, cancel = context.WithTimeout(context.Background(), *rule.Timeout)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), *p.config.PluginGlobalConfig.Timeout)
	}
	defer cancel()

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

	waitChan := make(chan struct{})
	defer close(waitChan)

	var m sync.Mutex
	timeout := false

	go func() {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return
			}
			klog.Errorf("Error in running plugin timeout %q", rule.Path)
			if cmd.Process == nil || cmd.Process.Pid == 0 {
				klog.Errorf("Error in cmd.Process check %q", rule.Path)
				break
			}

			m.Lock()
			timeout = true
			m.Unlock()

			err := util.Kill(cmd)
			if err != nil {
				klog.Errorf("Error in kill process %d, %v", cmd.Process.Pid, err)
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

	// Capture enough stdout to honor the configured max_output_length, bounded
	// by a hard safety ceiling. Previously this was a fixed 4 KiB buffer that
	// silently truncated plugins configured with a larger max_output_length.
	stdoutCapture := int64(*p.config.PluginGlobalConfig.MaxOutputLength)
	if stdoutCapture > maxCustomPluginStdoutCeilingBytes {
		stdoutCapture = maxCustomPluginStdoutCeilingBytes
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		stdout, stdoutErr = readFromReader(stdoutPipe, stdoutCapture)
	}()
	go func() {
		defer wg.Done()
		stderr, stderrErr = readFromReader(stderrPipe, maxCustomPluginStderrBytes)
	}()
	// This will wait for the reads to complete. If the execution times out, the pipes
	// will be closed and the wait group unblocks.
	wg.Wait()

	if stdoutErr != nil {
		klog.Errorf("Error reading stdout for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error reading stdout for plugin. Please check the error log"
	}

	if stderrErr != nil {
		klog.Errorf("Error reading stderr for plugin %q: error - %v", rule.Path, err)
		return cpmtypes.Unknown, "Error reading stderr for plugin. Please check the error log"
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			klog.Errorf("Error in waiting for plugin %q: error - %v. output - %q", rule.Path, err, string(stdout))
			return cpmtypes.Unknown, "Error in waiting for plugin. Please check the error log"
		}
	}

	// trim suffix useless bytes
	output = string(stdout)
	output = strings.TrimSpace(output)

	m.Lock()
	cmdKilled := timeout
	m.Unlock()

	if cmdKilled {
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
