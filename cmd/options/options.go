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

package options

import (
	"flag"
	"fmt"
	"os"
	"time"

	"net/url"

	"github.com/spf13/pflag"

	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/types"
)

// NodeProblemDetectorOptions contains node problem detector command line and application options.
type NodeProblemDetectorOptions struct {
	// command line options

	// PrintVersion is the flag determining whether version information is printed.
	PrintVersion bool
	// HostnameOverride specifies custom node name used to override hostname.
	HostnameOverride string
	// ServerPort is the port to bind the node problem detector server. Use 0 to disable.
	ServerPort int
	// ServerAddress is the address to bind the node problem detector server.
	ServerAddress string

	// exporter options

	// k8sExporter options
	// EnableK8sExporter is the flag determining whether to report to Kubernetes.
	EnableK8sExporter bool
	// ApiServerOverride is the custom URI used to connect to Kubernetes ApiServer.
	ApiServerOverride string
	// APIServerWaitTimeout is the timeout on waiting for kube-apiserver to be
	// ready.
	APIServerWaitTimeout time.Duration
	// APIServerWaitInterval is the interval between the checks on the
	// readiness of kube-apiserver.
	APIServerWaitInterval time.Duration

	// prometheusExporter options
	// PrometheusServerPort is the port to bind the Prometheus scrape endpoint. Use 0 to disable.
	PrometheusServerPort int
	// PrometheusServerAddress is the address to bind the Prometheus scrape endpoint.
	PrometheusServerAddress string

	// problem daemon options

	// SystemLogMonitorConfigPaths specifies the list of paths to system log monitor configuration
	// files.
	// SystemLogMonitorConfigPaths is used by the deprecated option --system-log-monitors. The new
	// option --config.system-log-monitor will stored the config file paths in MonitorConfigPaths.
	SystemLogMonitorConfigPaths []string
	// CustomPluginMonitorConfigPaths specifies the list of paths to custom plugin monitor configuration
	// files.
	// CustomPluginMonitorConfigPaths is used by the deprecated option --custom-plugin-monitors. The
	// new option --config.custom-plugin-monitor will stored the config file paths in MonitorConfigPaths.
	CustomPluginMonitorConfigPaths []string
	// MonitorConfigPaths specifies the list of paths to configuration files for each monitor.
	MonitorConfigPaths types.ProblemDaemonConfigPathMap

	// application options

	// NodeName is the node name used to communicate with Kubernetes ApiServer.
	NodeName string
}

func NewNodeProblemDetectorOptions() *NodeProblemDetectorOptions {
	npdo := &NodeProblemDetectorOptions{MonitorConfigPaths: types.ProblemDaemonConfigPathMap{}}
	for _, problemDaemonName := range problemdaemon.GetProblemDaemonNames() {
		npdo.MonitorConfigPaths[problemDaemonName] = &[]string{}
	}
	return npdo
}

// AddFlags adds node problem detector command line options to pflag.
func (npdo *NodeProblemDetectorOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&npdo.SystemLogMonitorConfigPaths, "system-log-monitors",
		[]string{}, "List of paths to system log monitor config files, comma separated.")
	fs.MarkDeprecated("system-log-monitors", "replaced by --config.system-log-monitor. NPD will panic if both --system-log-monitors and --config.system-log-monitor are set.")
	fs.StringSliceVar(&npdo.CustomPluginMonitorConfigPaths, "custom-plugin-monitors",
		[]string{}, "List of paths to custom plugin monitor config files, comma separated.")
	fs.MarkDeprecated("custom-plugin-monitors", "replaced by --config.custom-plugin-monitor. NPD will panic if both --custom-plugin-monitors and --config.custom-plugin-monitor are set.")
	fs.BoolVar(&npdo.EnableK8sExporter, "enable-k8s-exporter", true, "Enables reporting to Kubernetes API server.")
	fs.StringVar(&npdo.ApiServerOverride, "apiserver-override",
		"", "Custom URI used to connect to Kubernetes ApiServer. This is ignored if --enable-k8s-exporter is false.")
	fs.DurationVar(&npdo.APIServerWaitTimeout, "apiserver-wait-timeout", time.Duration(5)*time.Minute, "The timeout on waiting for kube-apiserver to be ready. This is ignored if --enable-k8s-exporter is false.")
	fs.DurationVar(&npdo.APIServerWaitInterval, "apiserver-wait-interval", time.Duration(5)*time.Second, "The interval between the checks on the readiness of kube-apiserver. This is ignored if --enable-k8s-exporter is false.")
	fs.BoolVar(&npdo.PrintVersion, "version", false, "Print version information and quit")
	fs.StringVar(&npdo.HostnameOverride, "hostname-override",
		"", "Custom node name used to override hostname")
	fs.IntVar(&npdo.ServerPort, "port",
		20256, "The port to bind the node problem detector server. Use 0 to disable.")
	fs.StringVar(&npdo.ServerAddress, "address",
		"127.0.0.1", "The address to bind the node problem detector server.")

	fs.IntVar(&npdo.PrometheusServerPort, "prometheus-port",
		20257, "The port to bind the Prometheus scrape endpoint. Prometheus exporter is enabled by default at port 20257. Use 0 to disable.")
	fs.StringVar(&npdo.PrometheusServerAddress, "prometheus-address",
		"127.0.0.1", "The address to bind the Prometheus scrape endpoint.")

	for _, problemDaemonName := range problemdaemon.GetProblemDaemonNames() {
		fs.StringSliceVar(
			npdo.MonitorConfigPaths[problemDaemonName],
			"config."+string(problemDaemonName),
			[]string{},
			fmt.Sprintf("Comma separated configurations for %v monitor. %v",
				problemDaemonName,
				problemdaemon.GetProblemDaemonHandlerOrDie(problemDaemonName).CmdOptionDescription))
	}
}

// ValidOrDie validates node problem detector command line options.
func (npdo *NodeProblemDetectorOptions) ValidOrDie() {
	if _, err := url.Parse(npdo.ApiServerOverride); npdo.EnableK8sExporter && err != nil {
		panic(fmt.Sprintf("apiserver-override %q is not a valid HTTP URI: %v",
			npdo.ApiServerOverride, err))
	}

	if len(npdo.SystemLogMonitorConfigPaths) != 0 {
		panic("SystemLogMonitorConfigPaths is deprecated. It should have been reassigned to MonitorConfigPaths. This should not happen.")
	}
	if len(npdo.CustomPluginMonitorConfigPaths) != 0 {
		panic("CustomPluginMonitorConfigPaths is deprecated. It should have been reassigned to MonitorConfigPaths. This should not happen.")
	}

	configCount := 0
	for _, problemDaemonConfigPaths := range npdo.MonitorConfigPaths {
		configCount += len(*problemDaemonConfigPaths)
	}
	if configCount == 0 {
		panic("No configuration option for any problem daemon is specified.")
	}
}

// Plugin names for custom plugin monitor and system log monitor.
// Hard code them here to:
// 1) Handle deprecated flags for --system-log-monitors and --custom-plugin-monitors.
// 2) Avoid direct dependencies to packages in those plugins, so that those plugins
// can be disabled at compile time.
const (
	customPluginMonitorName = "custom-plugin-monitor"
	systemLogMonitorName    = "system-log-monitor"
)

// SetConfigFromDeprecatedOptionsOrDie sets NPD option using deprecated options.
func (npdo *NodeProblemDetectorOptions) SetConfigFromDeprecatedOptionsOrDie() {
	if len(npdo.SystemLogMonitorConfigPaths) != 0 {
		if npdo.MonitorConfigPaths[systemLogMonitorName] == nil {
			// As long as the problem daemon is registered, MonitorConfigPaths should
			// not be nil.
			panic("System log monitor is not supported")
		}

		if len(*npdo.MonitorConfigPaths[systemLogMonitorName]) != 0 {
			panic("Option --system-log-monitors is deprecated in favor of --config.system-log-monitor. They cannot be set at the same time.")
		}

		*npdo.MonitorConfigPaths[systemLogMonitorName] = append(
			*npdo.MonitorConfigPaths[systemLogMonitorName],
			npdo.SystemLogMonitorConfigPaths...)
		npdo.SystemLogMonitorConfigPaths = []string{}
	}

	if len(npdo.CustomPluginMonitorConfigPaths) != 0 {
		if npdo.MonitorConfigPaths[customPluginMonitorName] == nil {
			// As long as the problem daemon is registered, MonitorConfigPaths should
			// not be nil.
			panic("Custom plugin monitor is not supported")
		}

		if len(*npdo.MonitorConfigPaths[customPluginMonitorName]) != 0 {
			panic("Option --custom-plugin-monitors is deprecated in favor of --config.custom-plugin-monitor. They cannot be set at the same time.")
		}

		*npdo.MonitorConfigPaths[customPluginMonitorName] = append(
			*npdo.MonitorConfigPaths[customPluginMonitorName],
			npdo.CustomPluginMonitorConfigPaths...)
		npdo.CustomPluginMonitorConfigPaths = []string{}
	}
}

// SetNodeNameOrDie sets `NodeName` field with valid value.
func (npdo *NodeProblemDetectorOptions) SetNodeNameOrDie() {
	// Check hostname override first for customized node name.
	if npdo.HostnameOverride != "" {
		npdo.NodeName = npdo.HostnameOverride
		return
	}

	// Get node name from environment variable NODE_NAME
	// By default, assume that the NODE_NAME env should have been set with
	// downward api or user defined exported environment variable. We prefer it because sometimes
	// the hostname returned by os.Hostname is not right because:
	// 1. User may override the hostname.
	// 2. For some cloud providers, os.Hostname is different from the real hostname.
	npdo.NodeName = os.Getenv("NODE_NAME")
	if npdo.NodeName != "" {
		return
	}

	// For backward compatibility. If the env is not set, get the hostname
	// from os.Hostname(). This may not work for all configurations and
	// environments.
	nodeName, err := os.Hostname()
	if err != nil {
		panic(fmt.Sprintf("Failed to get host name: %v", err))
	}

	npdo.NodeName = nodeName
}

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}
