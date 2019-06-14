/*
Copyright 2018 The Kubernetes Authors.

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

package options

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/types"
)

func equalMonitorConfigPaths(npdoX NodeProblemDetectorOptions, npdoY NodeProblemDetectorOptions) bool {
	monitorConfigPathsX, monitorConfigPathsY := npdoX.MonitorConfigPaths, npdoY.MonitorConfigPaths

	if monitorConfigPathsX == nil && monitorConfigPathsY == nil {
		return true
	}
	if monitorConfigPathsX == nil || monitorConfigPathsY == nil {
		return false
	}
	if len(monitorConfigPathsX) != len(monitorConfigPathsY) {
		return false
	}

	for problemDaemonType, configPathsX := range monitorConfigPathsX {
		configPathsY, ok := monitorConfigPathsY[problemDaemonType]
		if !ok {
			return false
		}
		if configPathsX == nil && configPathsY == nil {
			continue
		}
		if configPathsX == nil || configPathsY == nil {
			return false
		}
		if !reflect.DeepEqual(*configPathsX, *configPathsY) {
			return false
		}
	}
	return true
}

type options struct {
	Nodename         string
	HostnameOverride string
}

// TestSetNodeNameOrDie tests for permutations of nodename, hostname and hostnameoverride.
func TestSetNodeNameOrDie(t *testing.T) {
	hostName, err := os.Hostname()
	if err != nil {
		t.Errorf("Query hostname error: %v", err)
	}

	uts := map[string]struct {
		WantedNodeName string
		Meta           options
	}{
		"Check hostname override only": {
			WantedNodeName: "hostname-override",
			Meta: options{
				Nodename:         "node-name-env",
				HostnameOverride: "hostname-override",
			},
		},
		"Check hostname override and NODE_NAME env": {
			WantedNodeName: "node-name-env",
			Meta: options{
				Nodename:         "node-name-env",
				HostnameOverride: "",
			},
		},
		"Check hostname override, NODE_NAME env and hostname": {
			WantedNodeName: hostName,
			Meta: options{
				Nodename:         "",
				HostnameOverride: "",
			},
		},
	}

	for desc, ut := range uts {
		err := os.Unsetenv("NODE_NAME")
		if err != nil {
			t.Errorf("Desc: %v. Unset NODE_NAME env error: %v", desc, err)
		}

		if len(ut.Meta.Nodename) != 0 {
			err := os.Setenv("NODE_NAME", ut.Meta.Nodename)
			if err != nil {
				t.Errorf("Desc: %v. Set NODE_NAME env error: %v", desc, err)
			}
		}

		npdOpts := NewNodeProblemDetectorOptions()
		npdOpts.HostnameOverride = ut.Meta.HostnameOverride
		npdOpts.SetNodeNameOrDie()

		if npdOpts.NodeName != ut.WantedNodeName {
			t.Errorf("Desc: %v. Set node name error. Wanted: %v. Got: %v", desc, ut.WantedNodeName, npdOpts.NodeName)
		}
	}
}

func TestValidOrDie(t *testing.T) {
	fooMonitorConfigMap := types.ProblemDaemonConfigPathMap{}
	fooMonitorConfigMap["foo-monitor"] = &[]string{"config-a", "config-b"}

	emptyMonitorConfigMap := types.ProblemDaemonConfigPathMap{}

	testCases := []struct {
		name        string
		npdo        NodeProblemDetectorOptions
		expectPanic bool
	}{
		{
			name: "default k8s exporter config",
			npdo: NodeProblemDetectorOptions{
				MonitorConfigPaths: fooMonitorConfigMap,
			},
			expectPanic: false,
		},
		{
			name: "enables k8s exporter config",
			npdo: NodeProblemDetectorOptions{
				ApiServerOverride:  "",
				EnableK8sExporter:  true,
				MonitorConfigPaths: fooMonitorConfigMap,
			},
			expectPanic: false,
		},
		{
			name: "k8s exporter config with valid ApiServerOverride",
			npdo: NodeProblemDetectorOptions{
				ApiServerOverride:  "127.0.0.1",
				EnableK8sExporter:  true,
				MonitorConfigPaths: fooMonitorConfigMap,
			},
			expectPanic: false,
		},
		{
			name: "k8s exporter config with invalid ApiServerOverride",
			npdo: NodeProblemDetectorOptions{
				ApiServerOverride:  ":foo",
				EnableK8sExporter:  true,
				MonitorConfigPaths: fooMonitorConfigMap,
			},
			expectPanic: true,
		},
		{
			name: "non-empty MonitorConfigPaths",
			npdo: NodeProblemDetectorOptions{
				MonitorConfigPaths: fooMonitorConfigMap,
			},
			expectPanic: false,
		},
		{
			name: "empty MonitorConfigPaths",
			npdo: NodeProblemDetectorOptions{
				MonitorConfigPaths: emptyMonitorConfigMap,
			},
			expectPanic: true,
		},
		{
			name:        "un-initialized MonitorConfigPaths",
			npdo:        NodeProblemDetectorOptions{},
			expectPanic: true,
		},
		{
			name: "mixture of deprecated SystemLogMonitorConfigPaths and new MonitorConfigPaths",
			npdo: NodeProblemDetectorOptions{
				SystemLogMonitorConfigPaths: []string{"config-a"},
				MonitorConfigPaths:          fooMonitorConfigMap,
			},
			expectPanic: true,
		},
		{
			name: "mixture of deprecated CustomPluginMonitorConfigPaths and new MonitorConfigPaths",
			npdo: NodeProblemDetectorOptions{
				CustomPluginMonitorConfigPaths: []string{"config-a"},
				MonitorConfigPaths:             fooMonitorConfigMap,
			},
			expectPanic: true,
		},
		{
			name: "deprecated SystemLogMonitor option with empty MonitorConfigPaths",
			npdo: NodeProblemDetectorOptions{
				SystemLogMonitorConfigPaths: []string{"config-a"},
				MonitorConfigPaths:          emptyMonitorConfigMap,
			},
			expectPanic: true,
		},
		{
			name: "deprecated SystemLogMonitor option with un-initialized MonitorConfigPaths",
			npdo: NodeProblemDetectorOptions{
				SystemLogMonitorConfigPaths: []string{"config-a"},
			},
			expectPanic: true,
		},
		{
			name: "deprecated CustomPluginMonitor option with empty MonitorConfigPaths",
			npdo: NodeProblemDetectorOptions{
				CustomPluginMonitorConfigPaths: []string{"config-b"},
				MonitorConfigPaths:             emptyMonitorConfigMap,
			},
			expectPanic: true,
		},
		{
			name: "deprecated CustomPluginMonitor option with un-initialized MonitorConfigPaths",
			npdo: NodeProblemDetectorOptions{
				CustomPluginMonitorConfigPaths: []string{"config-b"},
			},
			expectPanic: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				assert.Panics(t, test.npdo.ValidOrDie, "NPD option %+v is invalid. Expected ValidOrDie to panic.", test.npdo)
			} else {
				assert.NotPanics(t, test.npdo.ValidOrDie, "NPD option %+v is valid. Expected ValidOrDie to not panic.", test.npdo)
			}
		})
	}
}

func TestSetConfigFromDeprecatedOptionsOrDie(t *testing.T) {
	testCases := []struct {
		name        string
		orig        NodeProblemDetectorOptions
		wanted      NodeProblemDetectorOptions
		expectPanic bool
	}{
		{
			name: "no deprecated options",
			orig: NodeProblemDetectorOptions{
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					systemLogMonitorName:    &[]string{"config-a", "config-b"},
					customPluginMonitorName: &[]string{"config-c", "config-d"},
				},
			},
			expectPanic: false,
			wanted: NodeProblemDetectorOptions{
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					systemLogMonitorName:    &[]string{"config-a", "config-b"},
					customPluginMonitorName: &[]string{"config-c", "config-d"},
				},
			},
		},
		{
			name: "correctly using deprecated options",
			orig: NodeProblemDetectorOptions{
				SystemLogMonitorConfigPaths:    []string{"config-a", "config-b"},
				CustomPluginMonitorConfigPaths: []string{"config-c", "config-d"},
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					customPluginMonitorName: &[]string{},
					systemLogMonitorName:    &[]string{},
				},
			},
			expectPanic: false,
			wanted: NodeProblemDetectorOptions{
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					systemLogMonitorName:    &[]string{"config-a", "config-b"},
					customPluginMonitorName: &[]string{"config-c", "config-d"},
				},
			},
		},
		{
			name: "using deprecated SystemLogMonitor option and new CustomPluginMonitor option",
			orig: NodeProblemDetectorOptions{
				SystemLogMonitorConfigPaths: []string{"config-a", "config-b"},
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					customPluginMonitorName: &[]string{"config-c", "config-d"},
					systemLogMonitorName:    &[]string{},
				},
			},
			expectPanic: false,
			wanted: NodeProblemDetectorOptions{
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					systemLogMonitorName:    &[]string{"config-a", "config-b"},
					customPluginMonitorName: &[]string{"config-c", "config-d"},
				},
			},
		},
		{
			name: "using deprecated CustomPluginMonitor option and new SystemLogMonitor option",
			orig: NodeProblemDetectorOptions{
				CustomPluginMonitorConfigPaths: []string{"config-a", "config-b"},
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					customPluginMonitorName: &[]string{},
					systemLogMonitorName:    &[]string{"config-c", "config-d"},
				},
			},
			expectPanic: false,
			wanted: NodeProblemDetectorOptions{
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					systemLogMonitorName:    &[]string{"config-c", "config-d"},
					customPluginMonitorName: &[]string{"config-a", "config-b"},
				},
			},
		},
		{
			name: "using deprecated & new options on SystemLogMonitor",
			orig: NodeProblemDetectorOptions{
				SystemLogMonitorConfigPaths: []string{"config-a"},
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					systemLogMonitorName: &[]string{"config-b"},
				},
			},
			expectPanic: true,
		},
		{
			name: "using deprecated & new options on CustomPluginMonitor",
			orig: NodeProblemDetectorOptions{
				CustomPluginMonitorConfigPaths: []string{"config-a"},
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					customPluginMonitorName: &[]string{"config-b"},
				},
			},
			expectPanic: true,
		},
		{
			name: "using deprecated options when SystemLogMonitor is not registered",
			orig: NodeProblemDetectorOptions{
				SystemLogMonitorConfigPaths:    []string{"config-a"},
				CustomPluginMonitorConfigPaths: []string{"config-b"},
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					customPluginMonitorName: &[]string{},
				},
			},
			expectPanic: true,
		},
		{
			name: "using deprecated options when CustomPluginMonitor is not registered",
			orig: NodeProblemDetectorOptions{
				SystemLogMonitorConfigPaths:    []string{"config-a"},
				CustomPluginMonitorConfigPaths: []string{"config-b"},
				MonitorConfigPaths: types.ProblemDaemonConfigPathMap{
					systemLogMonitorName: &[]string{},
				},
			},
			expectPanic: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				assert.Panics(t, test.orig.SetConfigFromDeprecatedOptionsOrDie,
					"NPD option %+v is illegal. Expected SetConfigFromDeprecatedOptionsOrDie to panic.", test.orig)
			} else {
				assert.NotPanics(t, test.orig.SetConfigFromDeprecatedOptionsOrDie,
					"NPD option %+v is illegal. Expected SetConfigFromDeprecatedOptionsOrDie to not panic.", test.orig)
				if !equalMonitorConfigPaths(test.orig, test.wanted) {
					t.Errorf("Expect to get NPD option %+v, but got %+v", test.wanted, test.orig)
				}
				assert.Len(t, test.orig.SystemLogMonitorConfigPaths, 0,
					"SystemLogMonitorConfigPaths is deprecated and should to be cleared.")
				assert.Len(t, test.orig.CustomPluginMonitorConfigPaths, 0,
					"CustomPluginMonitorConfigPaths is deprecated and should to be cleared.")
			}
		})
	}
}
