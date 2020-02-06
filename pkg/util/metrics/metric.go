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
package metrics

import (
	"sync"
)

const (
	CPURunnableTaskCountID  MetricID = "cpu/runnable_task_count"
	CPUUsageTimeID          MetricID = "cpu/usage_time"
	ProblemCounterID        MetricID = "problem_counter"
	ProblemGaugeID          MetricID = "problem_gauge"
	DiskIOTimeID            MetricID = "disk/io_time"
	DiskWeightedIOID        MetricID = "disk/weighted_io"
	DiskAvgQueueLenID       MetricID = "disk/avg_queue_len"
	DiskOpsCountID          MetricID = "disk/operation_count"
	DiskMergedOpsCountID    MetricID = "disk/merged_operation_count"
	DiskOpsBytesID          MetricID = "disk/operation_bytes_count"
	DiskOpsTimeID           MetricID = "disk/operation_time"
	DiskBytesUsedID         MetricID = "disk/bytes_used"
	HostUptimeID            MetricID = "host/uptime"
	MemoryBytesUsedID       MetricID = "memory/bytes_used"
	MemoryAnonymousUsedID   MetricID = "memory/anonymous_used"
	MemoryPageCacheUsedID   MetricID = "memory/page_cache_used"
	MemoryUnevictableUsedID MetricID = "memory/unevictable_used"
	MemoryDirtyUsedID       MetricID = "memory/dirty_used"
)

var MetricMap MetricMapping

func init() {
	MetricMap.mapMutex.Lock()
	defer MetricMap.mapMutex.Unlock()

	MetricMap.viewNameToMetricIDMap = make(map[string]MetricID)
}

type MetricID string

type MetricMapping struct {
	viewNameToMetricIDMap map[string]MetricID
	mapMutex              sync.RWMutex
}

func (mm *MetricMapping) AddMapping(metricID MetricID, viewName string) {
	mm.mapMutex.Lock()
	defer mm.mapMutex.Unlock()

	mm.viewNameToMetricIDMap[viewName] = metricID
}

func (mm *MetricMapping) ViewNameToMetricID(viewName string) (MetricID, bool) {
	mm.mapMutex.RLock()
	defer mm.mapMutex.RUnlock()

	id, ok := mm.viewNameToMetricIDMap[viewName]
	return id, ok
}
