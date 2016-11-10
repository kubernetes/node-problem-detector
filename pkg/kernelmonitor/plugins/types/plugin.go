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

package types

import (
	"io"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/types"
)

// Plugin is the interface of a log parsing plugin.
type Plugin interface {
	// ReadCloser is used to read logs.
	io.ReadCloser
	// Translate translates one log line into types.KernelLog.
	Translate(string) (*types.KernelLog, error)
}

// Config is the configuration of the plugin.
type Config struct {
	// Plugin is the name of plugin which is currently used.
	// Currently supported: syslog, journald.
	Plugin string `json:"plugin, omitempty"`
	// LogPath is the path to the log
	LogPath string `json:"logPath, omitempty"`
	// Lookback is the time kernel watcher looks up
	Lookback string `json:"lookback, omitempty"`
}

// PluginCreateFunc is the create function of a plugin.
type PluginCreateFunc func(Config) (Plugin, error)
