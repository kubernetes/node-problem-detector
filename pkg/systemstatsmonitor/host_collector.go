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
	"context"

	"github.com/golang/glog"
	"github.com/shirou/gopsutil/host"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type hostCollector struct {
	tags   []tag.Mutator
	uptime *stats.Int64Measure
}

func NewHostCollectorOrDie(hostConfig *ssmtypes.HostStatsConfig) *hostCollector {
	hc := hostCollector{}

	keyKernelVersion, err := tag.NewKey("kernel_version")
	if err != nil {
		glog.Fatalf("Failed to create kernel_version tag during initializing host collector: %v", err)
	}
	kernelVersion, err := host.KernelVersion()
	if err != nil {
		glog.Fatalf("Failed to retrieve kernel version: %v", err)
	}
	hc.tags = append(hc.tags, tag.Upsert(keyKernelVersion, kernelVersion))

	keyOSVersion, err := tag.NewKey("os_version")
	if err != nil {
		glog.Fatalf("Failed to create os_version tag during initializing host collector: %v", err)
	}
	osVersion, err := util.GetOSVersion()
	if err != nil {
		glog.Fatalf("Failed to retrieve OS version: %v", err)
	}
	hc.tags = append(hc.tags, tag.Upsert(keyOSVersion, osVersion))

	if hostConfig.MetricsConfigs["host/uptime"].DisplayName != "" {
		hc.uptime = metrics.NewInt64Metric(
			hostConfig.MetricsConfigs["host/uptime"].DisplayName,
			"The uptime of the operating system",
			"second",
			view.LastValue(),
			[]tag.Key{keyKernelVersion, keyOSVersion})
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

	if hc.uptime != nil {
		err := stats.RecordWithTags(context.Background(), hc.tags, hc.uptime.M(int64(uptime)))
		if err != nil {
			glog.Errorf("Failed to record current uptime (%d seconds) of the host: %v", uptime, err)
		}
	}
}
