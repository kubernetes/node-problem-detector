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
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
	cpmtypes "k8s.io/node-problem-detector/pkg/custompluginmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

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
	runTicker := time.NewTicker(*p.config.PluginGlobalConfig.InvokeInterval)

	for {
		select {
		case <-runTicker.C:
			glog.Info("Start to run custom plugins")

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
					end := time.Now()

					glog.V(3).Infof("Rule: %+v. Start time: %v. End time: %v. Duration: %v", rule, start, end, end.Sub(start))

					result := cpmtypes.Result{
						Rule:       rule,
						ExitStatus: exitStatus,
						Message:    message,
					}

					p.resultChan <- result

					glog.Infof("Add check result %+v for rule %+v", result, rule)
				}(rule)
			}

			p.Wait()
			glog.Info("Finish running custom plugins")
		case <-p.tomb.Stopping():
			glog.Info("Stopping plugin execution")
			p.tomb.Done()
		}
	}
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

	cmd := exec.CommandContext(ctx, rule.Path, rule.Args...)
	stdout, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			glog.Errorf("Error in running plugin %q: error - %v. output - %q", rule.Path, err, string(stdout))
			return cpmtypes.Unknown, "Error in running plugin. Please check the error log"
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
		return cpmtypes.OK, output
	case 1:
		return cpmtypes.NonOK, output
	default:
		return cpmtypes.Unknown, output
	}
}

func (p *Plugin) Stop() {
	p.tomb.Stop()
	glog.Info("Stop plugin execution")
}
