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

	"net/url"

	"github.com/spf13/pflag"
)

// NodeProblemDetectorOptions contains node problem detector command line and application options.
type NodeProblemDetectorOptions struct {
	// command line options

	// SystemLogMonitorConfigPaths specifies the list of paths to system log monitor configuration
	// files.
	SystemLogMonitorConfigPaths []string
	// CustomPluginMonitorConfigPaths specifies the list of paths to custom plugin monitor configuration
	// files.
	CustomPluginMonitorConfigPaths []string
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

	// application options

	// NodeName is the node name used to communicate with Kubernetes ApiServer.
	NodeName string
}

func NewNodeProblemDetectorOptions() *NodeProblemDetectorOptions {
	return &NodeProblemDetectorOptions{}
}

// AddFlags adds node problem detector command line options to pflag.
func (npdo *NodeProblemDetectorOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&npdo.SystemLogMonitorConfigPaths, "system-log-monitors",
		[]string{}, "List of paths to system log monitor config files, comma separated.")
	fs.StringSliceVar(&npdo.CustomPluginMonitorConfigPaths, "custom-plugin-monitors",
		[]string{}, "List of paths to custom plugin monitor config files, comma separated.")
	fs.BoolVar(&npdo.EnableK8sExporter, "enable-k8s-exporter", true, "Enables reporting to Kubernetes API server.")
	fs.StringVar(&npdo.ApiServerOverride, "apiserver-override",
		"", "Custom URI used to connect to Kubernetes ApiServer. This is ignored if --enable-k8s-exporter is false.")
	fs.BoolVar(&npdo.PrintVersion, "version", false, "Print version information and quit")
	fs.StringVar(&npdo.HostnameOverride, "hostname-override",
		"", "Custom node name used to override hostname")
	fs.IntVar(&npdo.ServerPort, "port",
		20256, "The port to bind the node problem detector server. Use 0 to disable.")
	fs.StringVar(&npdo.ServerAddress, "address",
		"127.0.0.1", "The address to bind the node problem detector server.")
}

// ValidOrDie validates node problem detector command line options.
func (npdo *NodeProblemDetectorOptions) ValidOrDie() {
	if _, err := url.Parse(npdo.ApiServerOverride); npdo.EnableK8sExporter && err != nil {
		panic(fmt.Sprintf("apiserver-override %q is not a valid HTTP URI: %v",
			npdo.ApiServerOverride, err))
	}
	if len(npdo.SystemLogMonitorConfigPaths) == 0 && len(npdo.CustomPluginMonitorConfigPaths) == 0 {
		panic(fmt.Sprintf("Either --system-log-monitors or --custom-plugin-monitors is required"))
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
