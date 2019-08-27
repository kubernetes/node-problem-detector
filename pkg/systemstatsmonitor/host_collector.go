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

package systemstatsmonitor

import (
	"github.com/golang/glog"
	"github.com/shirou/gopsutil/host"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type hostCollector struct {
	tags       map[string]string
	uptime     *metrics.Int64Metric
	lastUptime int64
}

func NewHostCollectorOrDie(hostConfig *ssmtypes.HostStatsConfig) *hostCollector {
	hc := hostCollector{map[string]string{}, nil, 0}

	kernelVersion, err := host.KernelVersion()
	if err != nil {
		glog.Fatalf("Failed to retrieve kernel version: %v", err)
	}
	hc.tags["kernel_version"] = kernelVersion

	osVersion, err := util.GetOSVersion()
	if err != nil {
		glog.Fatalf("Failed to retrieve OS version: %v", err)
	}
	hc.tags["os_version"] = osVersion

	// Use metrics.Sum aggregation method to ensure the metric is a counter/cumulative metric.
	if hostConfig.MetricsConfigs["host/uptime"].DisplayName != "" {
		hc.uptime, err = metrics.NewInt64Metric(
			hostConfig.MetricsConfigs["host/uptime"].DisplayName,
			"The uptime of the operating system",
			"second",
			metrics.Sum,
			[]string{"kernel_version", "os_version"})
		if err != nil {
			glog.Fatalf("Error initializing metric for host/uptime: %v", err)
		}
	}

	return &hc
}

func (hc *hostCollector) collect() {
	if hc == nil {
		return
	}

	uptime, err := host.Uptime()
	if err != nil {
		glog.Errorf("Failed to retrieve uptime of the host: %v", err)
		return
	}
	uptimeSeconds := int64(uptime)

	if hc.uptime != nil {
		hc.uptime.Record(hc.tags, uptimeSeconds-hc.lastUptime)
	}
	hc.lastUptime = uptimeSeconds
}
