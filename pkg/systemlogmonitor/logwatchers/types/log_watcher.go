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
	"k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

// LogWatcher is the interface of a log watcher.
type LogWatcher interface {
	// Watch starts watching logs and returns logs via a channel.
	Watch() (<-chan *types.Log, error)
	// Stop stops the log watcher. Resources open should be closed properly.
	Stop()
}

// WatcherConfig is the configuration of the log watcher.
type WatcherConfig struct {
	// Plugin is the name of plugin which is currently used.
	// Currently supported: filelog, journald, kmsg.
	Plugin string `json:"plugin,omitempty"`
	// PluginConfig is a key/value configuration of a plugin. Valid configurations
	// are defined in different log watcher plugin.
	PluginConfig map[string]string `json:"pluginConfig,omitempty"`
	// LogPath is the path to the log
	LogPath string `json:"logPath,omitempty"`
	// Lookback is the time log watcher looks up
	Lookback string `json:"lookback,omitempty"`
	// Delay is the time duration log watcher delays after node boot time. This is
	// useful when the log watcher needs to wait for some time until the node
	// becomes stable.
	Delay string `json:"delay,omitempty"`
}

// WatcherCreateFunc is the create function of a log watcher.
type WatcherCreateFunc func(WatcherConfig) LogWatcher
