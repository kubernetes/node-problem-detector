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

package types

import (
	"github.com/spf13/pflag"
)

type CommandLineOptions struct {
	// DeprecatedCustomPluginMonitorConfigPaths specifies the list of paths to custom plugin monitor configuration
	// files. DeprecatedCustomPluginMonitorConfigPaths is used by the deprecated option --custom-plugin-monitors.
	DeprecatedCustomPluginMonitorConfigPaths []string
	// CustomPluginMonitorConfigPaths specifies the list of paths to custom plugin monitor configuration
	// files. CustomPluginMonitorConfigPaths is used by the option --config.custom-plugin-monitors.
	CustomPluginMonitorConfigPaths []string
}

func (clo *CommandLineOptions) SetFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&clo.DeprecatedCustomPluginMonitorConfigPaths, "custom-plugin-monitors",
		[]string{}, "List of paths to custom plugin monitor config files, comma separated.")
	fs.MarkDeprecated("custom-plugin-monitors",
		"replaced by --config.custom-plugin-monitor. NPD will panic if both --custom-plugin-monitors and --config.custom-plugin-monitor are set.")

	fs.StringSliceVar(&clo.CustomPluginMonitorConfigPaths, "config.custom-plugin-monitor",
		[]string{}, "List of paths to custom plugin monitor config files, comma separated.")
}
