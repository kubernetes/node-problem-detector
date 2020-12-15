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

package systemstatsmonitor

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/golang/glog"
	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
	"k8s.io/node-problem-detector/pkg/util/metrics/system"
)

type osFeatureCollector struct {
	config    *ssmtypes.OSFeatureStatsConfig
	osFeature *metrics.Int64Metric
}

func NewOsFeatureCollectorOrDie(osFeatureConfig *ssmtypes.OSFeatureStatsConfig) *osFeatureCollector {
	oc := osFeatureCollector{config: osFeatureConfig}
	var err error
	// Use metrics.Last aggregation method to ensure the metric is a guage metric.
	if osFeatureConfig.MetricsConfigs["system/os_feature"].DisplayName != "" {
		oc.osFeature, err = metrics.NewInt64Metric(
			metrics.OSFeatureID,
			osFeatureConfig.MetricsConfigs[string(metrics.OSFeatureID)].DisplayName,
			"OS Features like GPU support, KTD kernel, third party modules as unknown modules. 1 if the feature is enabled and 0, if disabled.",
			"1",
			metrics.LastValue,
			[]string{featureLabel, valueLabel})
		if err != nil {
			glog.Fatalf("Error initializing metric for system/os_feature: %v", err)
		}
	}
	return &oc
}

// recordFeaturesFromCmdline records the guest OS features that can be derived
// from the /proc/cmdline
// The following features are recorded:
// 1. KTD kernel based on csm.enabled value
// 2. UnifiedCgroupHierarchy based on systemd.unified_cgroup_hierarchy
// 3. KernelModuleIntegrity based on the loadpin enabled and a module signed.
func (ofc *osFeatureCollector) recordFeaturesFromCmdline(cmdlineArgs []system.CmdlineArg) {
	var featuresMap = map[string]int64{
		"KTD":                    0,
		"UnifiedCgroupHierarchy": 0,
		"ModuleSigned":           0,
		"LoadPinEnabled":         0,
	}
	for _, cmdlineArg := range cmdlineArgs {
		// record KTD feature.
		if cmdlineArg.Key == "csm.enabled" {
			featuresMap["KTD"], _ = strconv.ParseInt(cmdlineArg.Value, 10, 64)
		}
		// record UnifiedCgroupHierarchy feature.
		if cmdlineArg.Key == "systemd.unified_cgroup_hierarchy" {
			featuresMap["UnifiedCgroupHierarchy"], _ = strconv.ParseInt(cmdlineArg.Value, 10, 64)
		}
		// record KernelModuleIntegrity feature.
		if cmdlineArg.Key == "module.sig_enforce" {
			featuresMap["ModuleSigned"], _ = strconv.ParseInt(cmdlineArg.Value, 10, 64)
		}
		if cmdlineArg.Key == "loadpin.enabled" {
			featuresMap["LoadPinEnabled"], _ = strconv.ParseInt(cmdlineArg.Value, 10, 64)
		}
	}
	// Record the feature values.
	ofc.osFeature.Record(map[string]string{featureLabel: "KTD"}, featuresMap["KTD"])
	ofc.osFeature.Record(map[string]string{featureLabel: "UnifiedCgroupHierarchy"}, featuresMap["UnifiedCgroupHierarchy"])
	if featuresMap["ModuleSigned"] == 1 && featuresMap["LoadPinEnabled"] == 1 {
		ofc.osFeature.Record(map[string]string{featureLabel: "KernelModuleIntegrity"}, 1)
	} else {
		ofc.osFeature.Record(map[string]string{featureLabel: "KernelModuleIntegrity"}, 0)
	}
}

// recordFeaturesFromCmdline records the guest OS features that can be derived
// from the /proc/modules
// The following features are recorded:
// 1. GPUSupport based on the presence of nvidia module
// 2. UnknownModules are tracked based on the presence of thirdparty kernel modules.
func (ofc *osFeatureCollector) recordFeaturesFromModules(modules []system.Module) {
	// Collect known modules (default modules based on guest OS present in known-modules.json)
	var knownModules []system.Module
	f, err := ioutil.ReadFile(ofc.config.KnownModulesConfigPath)
	if err != nil {
		glog.Warningf("Failed to read configuration file %s: %v",
			ofc.config.KnownModulesConfigPath, err)
	}
	// When the knownModulesConfigPath is not set
	// it should assume all the metrics are assumed to be default modules.
	if f != nil {
		err = json.Unmarshal(f, &knownModules)
		if err != nil {
			glog.Warningf("Failed to retrieve known modules %v", err)
		}
	} else {
		knownModules = []system.Module{}
	}

	var hasGPUSupport = 0
	unknownModules := []string{}

	// Collect UnknownModules and check GPUSupport
	for _, module := range modules {
		// if the module has nvidia modules, then the hasGPUSupport is set.
		if strings.Contains(module.ModuleName, "nvidia") {
			hasGPUSupport = 1
		} else {
			if module.OutOfTree || module.Proprietary {
				if !system.ContainsModule(module.ModuleName, knownModules) {
					unknownModules = append(unknownModules, module.ModuleName)
				}
			}
		}
	}
	// record the UnknownModules and GPUSupport
	if len(unknownModules) > 0 {
		ofc.osFeature.Record(map[string]string{featureLabel: "UnknownModules",
			valueLabel: strings.Join(unknownModules, ",")}, 1)
	} else {
		ofc.osFeature.Record(map[string]string{featureLabel: "UnknownModules"},
			0)
	}
	ofc.osFeature.Record(map[string]string{featureLabel: "GPUSupport"},
		int64(hasGPUSupport))
}

func (ofc *osFeatureCollector) collect() {
	cmdlineArgs, err := system.CmdlineArgs()
	if err != nil {
		glog.Fatalf("Error retrieving cmdline args: %v", err)
	}
	ofc.recordFeaturesFromCmdline(cmdlineArgs)
	modules, err := system.Modules()
	if err != nil {
		glog.Fatalf("Error retrieving kernel modules: %v", err)
	}
	ofc.recordFeaturesFromModules(modules)
}
