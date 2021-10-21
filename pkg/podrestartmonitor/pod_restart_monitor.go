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

package podrestartmonitor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/types"
)

const PodRestartMonitorName = "pod-restart-monitor"

func init() {
	problemdaemon.Register(
		PodRestartMonitorName,
		types.ProblemDaemonHandler{
			CreateProblemDaemonOrDie: NewMonitorOrDie,
			CmdOptionDescription:     "Set to config file paths."})
}

type podRestartMonitor struct {
	configPath string
	config     MonitorConfig
	stop       chan struct{}
	clientset  *kubernetes.Clientset
	statuses   chan *types.Status
}

// NewMonitorOrDie create a new podRestartMonitor, panic if error occurs.
func NewMonitorOrDie(configPath string) types.Monitor {
	l := &podRestartMonitor{
		configPath: configPath,
	}

	f, err := ioutil.ReadFile(configPath)
	if err != nil {
		glog.Fatalf("Failed to read configuration file %q: %v", configPath, err)
	}
	if l.config, err = parseConfig(f); err != nil {
		glog.Fatalf("Failed to parse configuration file %q: %v", configPath, err)
	}

	glog.Infof("Finish parsing pod restart monitor config file %s: %+v", l.configPath, l.config)
	l.stop = make(chan struct{})

	return l
}

func parseConfig(f []byte) (config MonitorConfig, err error) {
	if err = json.Unmarshal(f, &config); err != nil {
		return config, err
	}

	// Apply default configurations
	if config.CheckInterval == 0 {
		config.CheckInterval = 5 * time.Minute
	}
	if config.RestartThreshold == 0 {
		config.RestartThreshold = 5
	}
	if config.Namespace == "" {
		config.Namespace = "kube-system"
	}
	if config.PodSelector == "" {
		return config, fmt.Errorf("did not specify a PodSelector")
	}
	if config.ConditionName == "" {
		config.ConditionName = ConditionTooManyRestarts
	}
	if _, err := labels.Parse(config.PodSelector); err != nil {
		return config, fmt.Errorf("invalid PodSelector %q: %w", config.PodSelector, err)
	}

	return config, nil
}

func (l *podRestartMonitor) Start() (<-chan *types.Status, error) {
	glog.Infof("Start pod restart monitor %s", l.configPath)

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	l.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	l.statuses = make(chan *types.Status)

	go l.monitorLoop()

	return l.statuses, nil
}

func getNodeName() string {
	return os.Getenv("NODE_NAME")
}

func (l *podRestartMonitor) Stop() {
	glog.Infof("Stopping monitor %s", l.configPath)
	close(l.stop)
}

// ConditionTooManyRestarts indicates that a Pod has restarted too many times, indicating a problem with the node.
const ConditionTooManyRestarts = "TooManyRestarts"

// monitorLoop is the main loop of log monitor.
func (l *podRestartMonitor) monitorLoop() {
	nodeName := getNodeName()
	conditionType := string(l.config.ConditionName)
MainLoop:
	for {
		select {
		case <-l.stop:
			glog.Warningf("monitor stopped")
			return
		case <-time.After(time.Duration(l.config.CheckInterval)):
			list, err := l.clientset.CoreV1().Pods(l.config.Namespace).List(metav1.ListOptions{
				LabelSelector: l.config.PodSelector,
				FieldSelector: "spec.nodeName=" + nodeName,
			})
			if err != nil {
				glog.Errorf("failed to list pods: %+v", err)
				continue
			}
			for _, pod := range list.Items {
				for _, c := range pod.Status.ContainerStatuses {
					count := c.RestartCount
					if count > l.config.RestartThreshold {
						l.statuses <- &types.Status{
							Source: PodRestartMonitorName,
							Events: nil,
							Conditions: []types.Condition{
								{Type: ConditionTooManyRestarts, Status: types.True, Transition: time.Now(),
									Reason: pod.Name, Message: fmt.Sprintf("%s has restarted %d times on %s", pod.Name, count, nodeName)},
							},
						}
						// we've marked the node, now we're done
						continue MainLoop
					}
				}
			}
			node, err := l.clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
			if err != nil {
				glog.Errorf("failed to get node status: %+v", err)
				continue
			}
			if hasCondition(node.Status.Conditions, ConditionTooManyRestarts) {
				l.statuses <- &types.Status{
					Source: PodRestartMonitorName,
					Events: nil,
					Conditions: []types.Condition{
						{Type: conditionType, Status: types.False, Transition: time.Now(),
							Reason: "Pod was removed",
							Message: fmt.Sprintf("No pod matching %s has restarted >%d times.",
								l.config.PodSelector, l.config.RestartThreshold)},
					},
				}
			}
		}
	}
}

func hasCondition(conditions []v1.NodeCondition, condition v1.NodeConditionType) bool {
	for _, nodeCondition := range conditions {
		if nodeCondition.Type == condition {
			return nodeCondition.Status == v1.ConditionTrue
		}
	}
	return false
}
