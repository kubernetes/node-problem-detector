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

package logwatchers

import (
	"k8s.io/node-problem-detector/pkg/systemlogmonitor/logwatchers/types"

	"github.com/golang/glog"
)

// createFuncs is a table of createFuncs for all supported log watchers.
var createFuncs = map[string]types.WatcherCreateFunc{}

// registerLogWatcher registers a createFunc for a log watcher.
func registerLogWatcher(name string, create types.WatcherCreateFunc) {
	createFuncs[name] = create
}

// GetLogWatcherOrDie get a log watcher based on the passed in configuration.
// The function panics when encounters an error.
func GetLogWatcherOrDie(config types.WatcherConfig) types.LogWatcher {
	create, ok := createFuncs[config.Plugin]
	if !ok {
		glog.Fatalf("No create function found for plugin %q", config.Plugin)
	}
	glog.Infof("Use log watcher of plugin %q", config.Plugin)
	return create(config)
}
