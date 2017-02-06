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

	"github.com/spf13/pflag"
)

type NodeProblemDetectorOptions struct {
	KernelMonitorConfigPath string
	ApiServerOverride       string
	PrintVersion            bool
	HostnameOverride        string
	ServerPort              int
	ServerAddress           string
}

func NewNodeProblemDetectorOptions() *NodeProblemDetectorOptions {
	return &NodeProblemDetectorOptions{}
}

func (npdo *NodeProblemDetectorOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&npdo.KernelMonitorConfigPath, "kernel-monitor",
		"/config/kernel-monitor.json", "The path to the kernel monitor config file")
	fs.StringVar(&npdo.ApiServerOverride, "apiserver-override",
		"", "Custom URI used to connect to Kubernetes ApiServer")
	fs.BoolVar(&npdo.PrintVersion, "version", false, "Print version information and quit")
	fs.StringVar(&npdo.HostnameOverride, "hostname-override",
		"", "Custom node name used to override hostname")
	fs.IntVar(&npdo.ServerPort, "port",
		10256, "The port to bind the node problem detector server. Use 0 to disable.")
	fs.StringVar(&npdo.ServerAddress, "address",
		"127.0.0.1", "The address to bind the node problem detector server.")
}

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}
