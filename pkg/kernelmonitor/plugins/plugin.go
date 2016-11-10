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

package plugins

import (
	"fmt"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/plugins/types"

	"github.com/golang/glog"
)

// createFuncs is a table of createFuncs for all supported plugin.
var createFuncs = map[string]types.PluginCreateFunc{}

// regsiterPlugin register a createFunc for a plugin.
func regsiterPlugin(name string, create types.PluginCreateFunc) {
	createFuncs[name] = create
}

// GetPlugin get a plugin based on the passed in configuration.
func GetPlugin(config types.Config) (types.Plugin, error) {
	create, ok := createFuncs[config.Plugin]
	if !ok {
		return nil, fmt.Errorf("no create function found for plugin %q", config.Plugin)
	}
	glog.Infof("Use log parsing plugin %q", config.Plugin)
	return create(config)
}
