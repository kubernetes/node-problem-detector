/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package systemlogmonitor

import (
	"github.com/spf13/pflag"
)

type commandLineOptions struct {
	// DeprecatedSystemLogMonitorConfigPaths specifies the list of paths to system log monitor configuration
	// files. DeprecatedSystemLogMonitorConfigPaths is used by the deprecated option --system-log-monitors.
	DeprecatedSystemLogMonitorConfigPaths []string
	// SystemLogMonitorConfigPaths specifies the list of paths to system log monitor configuration
	// files. SystemLogMonitorConfigPaths is used by the option --config.system-log-monitors.
	SystemLogMonitorConfigPaths []string
}

func (clo *commandLineOptions) SetFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&clo.DeprecatedSystemLogMonitorConfigPaths, "system-log-monitors",
		[]string{}, "List of paths to system log monitor config files, comma separated.")
	fs.MarkDeprecated("system-log-monitors", "replaced by --config.system-log-monitor. NPD will panic if both --system-log-monitors and --config.system-log-monitor are set.")

	fs.StringSliceVar(&clo.SystemLogMonitorConfigPaths, "config.system-log-monitor",
		[]string{}, "List of paths to system log monitor config files, comma separated.")
}
