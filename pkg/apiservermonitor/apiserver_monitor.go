/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package apiservermonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	kube_rest "k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemmetrics"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

// APIServerMonitorName is the name of the apiserver problem daemon.
const APIServerMonitorName = "apiserver-monitor"

type apiServerMonitor struct {
	configPath string
	config     MonitorConfig
	kubeConfig *kube_rest.Config
	conditions []types.Condition
	statusChan chan *types.Status
	tomb       *tomb.Tomb
}

func init() {
	problemdaemon.Register(
		APIServerMonitorName,
		types.ProblemDaemonHandler{
			CreateProblemDaemonOrDie: NewAPIServerMonitorOrDie,
			CmdOptionDescription:     "Set to config file paths.",
		})
}

// NewAPIServerMonitorOrDie creates a new apiServerMonitor, panics or exits if an error occurs.
func NewAPIServerMonitorOrDie(configPath string) types.Monitor {
	a := &apiServerMonitor{
		configPath: configPath,
		tomb:       tomb.NewTomb(),
	}

	f, err := os.ReadFile(configPath)
	if err != nil {
		klog.Fatalf("Failed to read configuration file %q: %v", configPath, err)
	}
	if err := json.Unmarshal(f, &a.config); err != nil {
		klog.Fatalf("Failed to unmarshal configuration file %q: %v", configPath, err)
	}
	if err := a.config.ApplyConfiguration(); err != nil {
		klog.Fatalf("Failed to apply configuration for %q: %v", configPath, err)
	}
	if err := a.config.Validate(); err != nil {
		klog.Fatalf("Failed to validate apiserver monitor config %+v: %v", a.config, err)
	}

	klog.Infof("Finish parsing apiserver monitor config file %s: %+v", a.configPath, a.config)

	if util.GlobalKubeClientConfig == nil {
		klog.Fatalf("GlobalKubeClientConfig is not initialized")
	}
	kubeConfig := kube_rest.CopyConfig(util.GlobalKubeClientConfig)
	kubeConfig.Timeout = a.config.Timeout
	if kubeConfig.Proxy == nil {
		// Explicitly set Proxy so client-go's tlsTransportCache treats this config as
		// uncacheable and creates a fresh http.Transport on each RESTClientFor call
		// instead of reusing a cached transport and its already-established TCP connection.
		kubeConfig.Proxy = http.ProxyFromEnvironment
	}
	a.kubeConfig = kubeConfig

	// A 1000 size channel should be big enough.
	a.statusChan = make(chan *types.Status, 1000)

	if *a.config.EnableMetricsReporting {
		initializeProblemMetricsOrDie(a.config.DefaultConditions)
	}
	return a
}

// initializeProblemMetricsOrDie creates problem metrics for all configured conditions and sets the initial value to 0.
func initializeProblemMetricsOrDie(conditions []types.Condition) {
	if problemmetrics.GlobalProblemMetricsManager == nil {
		return
	}
	for _, cond := range conditions {
		reason := cond.Type
		if err := problemmetrics.GlobalProblemMetricsManager.SetProblemGauge(cond.Type, reason, false); err != nil {
			klog.Fatalf("Failed to initialize problem gauge metrics for problem %q, reason %q: %v",
				cond.Type, reason, err)
		}
		if err := problemmetrics.GlobalProblemMetricsManager.IncrementProblemCounter(reason, 0); err != nil {
			klog.Fatalf("Failed to initialize problem counter metrics for %q: %v", reason, err)
		}
	}
}

func (a *apiServerMonitor) Start() (<-chan *types.Status, error) {
	klog.Infof("Start apiserver monitor %s", a.configPath)
	go a.monitorLoop()
	return a.statusChan, nil
}

func (a *apiServerMonitor) Stop() {
	klog.Infof("Stop apiserver monitor %s", a.configPath)
	a.tomb.Stop()
}

func (a *apiServerMonitor) monitorLoop() {
	defer a.tomb.Done()

	a.initializeConditions()
	if *a.config.SkipInitialStatus {
		klog.Infof("Skipping sending initial status. Using default conditions: %+v", a.conditions)
	} else {
		a.sendInitialStatus()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-a.tomb.Stopping():
			cancel()
		case <-ctx.Done():
		}
	}()

	runTicker := time.NewTicker(a.config.InvokeInterval)
	defer runTicker.Stop()

	select {
	case <-a.tomb.Stopping():
		klog.Infof("APIServer monitor stopped: %s", a.configPath)
		return
	default:
		a.runCheck(ctx)
	}

	for {
		select {
		case <-runTicker.C:
			a.runCheck(ctx)
		case <-a.tomb.Stopping():
			klog.Infof("APIServer monitor stopped: %s", a.configPath)
			return
		}
	}
}

func (a *apiServerMonitor) runCheck(ctx context.Context) {
	err := a.checkAPIServer(ctx)
	select {
	case <-a.tomb.Stopping():
		return
	default:
	}
	status := a.generateStatus(err)
	klog.V(3).Infof("New status generated: %+v", status)
	a.statusChan <- status
}

func (a *apiServerMonitor) checkAPIServer(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	client, err := kube_rest.RESTClientFor(a.kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}
	if client.Client != nil {
		defer utilnet.CloseIdleConnectionsFor(client.Client.Transport)
	}
	return client.Get().AbsPath("/version").Do(checkCtx).Error()
}

func (a *apiServerMonitor) generateStatus(checkErr error) *types.Status {
	timestamp := time.Now()
	var activeProblemEvents []types.Event
	var inactiveProblemEvents []types.Event

	for i := range a.conditions {
		condition := &a.conditions[i]
		defaultCondition := a.config.DefaultConditions[i]

		newStatus := types.False
		newReason := defaultCondition.Reason
		newMessage := defaultCondition.Message
		if checkErr != nil {
			newStatus = types.True
			newReason = condition.Type
			newMessage = checkErr.Error()
		}

		if condition.Status != newStatus {
			condition.Transition = timestamp
			condition.Status = newStatus
			condition.Reason = newReason
			condition.Message = newMessage

			updateEvent := util.GenerateConditionChangeEvent(
				condition.Type,
				newStatus,
				newReason,
				newMessage,
				timestamp,
			)
			if newStatus == types.True {
				activeProblemEvents = append(activeProblemEvents, updateEvent)
			} else {
				inactiveProblemEvents = append(inactiveProblemEvents, updateEvent)
			}
		}
	}

	if *a.config.EnableMetricsReporting && problemmetrics.GlobalProblemMetricsManager != nil {
		for _, event := range activeProblemEvents {
			if err := problemmetrics.GlobalProblemMetricsManager.IncrementProblemCounter(event.Reason, 1); err != nil {
				klog.Errorf("Failed to update problem counter metrics for %q: %v", event.Reason, err)
			}
		}
		for _, condition := range a.conditions {
			if err := problemmetrics.GlobalProblemMetricsManager.SetProblemGauge(
				condition.Type, condition.Reason, condition.Status == types.True); err != nil {
				klog.Errorf("Failed to update problem gauge metrics for problem %q, reason %q: %v",
					condition.Type, condition.Reason, err)
			}
		}
	}

	status := &types.Status{
		Source:     a.config.Source,
		Events:     append(activeProblemEvents, inactiveProblemEvents...),
		Conditions: slices.Clone(a.conditions),
	}
	if len(activeProblemEvents) != 0 || len(inactiveProblemEvents) != 0 {
		klog.V(0).Infof("New status generated: %+v", status)
	}
	return status
}

func (a *apiServerMonitor) sendInitialStatus() {
	klog.Infof("Sending initial status for %s with conditions: %+v", a.config.Source, a.conditions)
	a.statusChan <- &types.Status{
		Source:     a.config.Source,
		Conditions: slices.Clone(a.conditions),
	}
}

func (a *apiServerMonitor) initializeConditions() {
	a.conditions = initialConditions(a.config.DefaultConditions)
	klog.Infof("Initialized conditions for %s: %+v", a.configPath, a.conditions)
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
