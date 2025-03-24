/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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
	"runtime"
	"time"

	"github.com/spf13/pflag"

	"k8s.io/node-problem-detector/pkg/healthchecker/types"
)

// NewHealthCheckerOptions returns an empty health check options struct.
func NewHealthCheckerOptions() *HealthCheckerOptions {
	return &HealthCheckerOptions{}
}

// HealthCheckerOptions are the options used to configure the health checker.
type HealthCheckerOptions struct {
	Component          string
	Service            string
	EnableRepair       bool
	CriCtlPath         string
	CriSocketPath      string
	CriTimeout         time.Duration
	CoolDownTime       time.Duration
	LoopBackTime       time.Duration
	HealthCheckTimeout time.Duration
	LogPatterns        types.LogPatternFlag
}

// AddFlags adds health checker command line options to pflag.
func (hco *HealthCheckerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&hco.Component, "component", types.KubeletComponent,
		"The component to check health for. Supports kubelet, kube-proxy, and cri")
	// Deprecated: For backward compatibility on linux environment. Going forward "service" will be used instead of systemd-service
	if runtime.GOOS == "linux" {
		fs.MarkDeprecated("systemd-service", "please use --service flag instead")
		fs.StringVar(&hco.Service, "systemd-service", "",
			"The underlying service responsible for the component. Set to the corresponding component for kubelet, containerd for cri.")
	}
	fs.StringVar(&hco.Service, "service", "",
		"The underlying service responsible for the component. Set to the corresponding component for kubelet, containerd for cri.")
	fs.BoolVar(&hco.EnableRepair, "enable-repair", true, "Flag to enable/disable repair attempt for the component.")
	fs.StringVar(&hco.CriCtlPath, "crictl-path", types.DefaultCriCtl,
		"The path to the crictl binary. This is used to check health of cri component.")
	fs.StringVar(&hco.CriSocketPath, "cri-socket-path", types.DefaultCriSocketPath,
		"The path to the cri socket. Used with crictl to specify the socket path.")
	fs.DurationVar(&hco.CriTimeout, "cri-timeout", types.DefaultCriTimeout,
		"The duration to wait for crictl to run.")
	fs.DurationVar(&hco.CoolDownTime, "cooldown-time", types.DefaultCoolDownTime,
		"The duration to wait for the service to be up before attempting repair.")
	fs.DurationVar(&hco.LoopBackTime, "loopback-time", types.DefaultLoopBackTime,
		"The duration to loop back, if it is 0, health-check will check from start time.")
	fs.DurationVar(&hco.HealthCheckTimeout, "health-check-timeout", types.DefaultHealthCheckTimeout,
		"The time to wait before marking the component as unhealthy.")
	fs.Var(&hco.LogPatterns, "log-pattern",
		"The log pattern to look for in service journald logs. The format for flag value <failureThresholdCount>:<logPattern>")
}

// IsValid validates health checker command line options.
// Returns error if invalid, nil otherwise.
func (hco *HealthCheckerOptions) IsValid() error {
	// Make sure the component specified is valid.
	if hco.Component != types.KubeletComponent && hco.Component != types.CRIComponent && hco.Component != types.KubeProxyComponent {
		return fmt.Errorf("the component specified is not supported. Supported components are : <kubelet/cri/kube-proxy>")
	}
	// Make sure the service is specified if repair is enabled.
	if hco.EnableRepair && hco.Service == "" {
		return fmt.Errorf("service cannot be empty when repair is enabled")
	}
	// Skip checking further if the component is not cri.
	if hco.Component != types.CRIComponent {
		return nil
	}
	// Make sure the crictl path is not empty for cri component.
	if hco.Component == types.CRIComponent && hco.CriCtlPath == "" {
		return fmt.Errorf("the crictl-path cannot be empty for cri component")
	}
	// Make sure the cri socker path is not empty for cri component.
	if hco.Component == types.CRIComponent && hco.CriSocketPath == "" {
		return fmt.Errorf("the cri-socket-path cannot be empty for cri component")
	}
	return nil
}

// SetDefaults sets the defaults values for the dependent flags.
func (hco *HealthCheckerOptions) SetDefaults() {
	if hco.Service != "" {
		return
	}
	if hco.Component != types.CRIComponent {
		hco.Service = hco.Component
		return
	}
	hco.Service = types.ContainerdService
}

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
}
