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
	"k8s.io/klog/v2"

	"github.com/shirou/gopsutil/v3/mem"
)

func (mc *memoryCollector) collect() {
	if mc == nil {
		return
	}

	meminfo, err := mem.VirtualMemory()
	if err != nil {
		klog.Errorf("cannot get windows memory metrics from GlobalMemoryStatusEx: %v", err)
		return
	}

	if mc.mBytesUsed != nil {
		mc.mBytesUsed.Record(map[string]string{stateLabel: "free"}, int64(meminfo.Available)*1024)
		mc.mBytesUsed.Record(map[string]string{stateLabel: "used"}, int64(meminfo.Used)*1024)
	}
}
