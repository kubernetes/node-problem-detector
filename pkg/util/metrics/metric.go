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
	CPULoad1m               MetricID = "cpu/load_1m"
	CPULoad5m               MetricID = "cpu/load_5m"
	CPULoad15m              MetricID = "cpu/load_15m"
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
	DiskPercentUsedID       MetricID = "disk/percent_used"
	HostUptimeID            MetricID = "host/uptime"
	MemoryBytesUsedID       MetricID = "memory/bytes_used"
	MemoryAnonymousUsedID   MetricID = "memory/anonymous_used"
	MemoryPageCacheUsedID   MetricID = "memory/page_cache_used"
	MemoryUnevictableUsedID MetricID = "memory/unevictable_used"
	MemoryDirtyUsedID       MetricID = "memory/dirty_used"
	MemoryPercentUsedID     MetricID = "memory/percent_used"
	OSFeatureID             MetricID = "system/os_feature"
	SystemProcessesTotal    MetricID = "system/processes_total"
	SystemProcsRunning      MetricID = "system/procs_running"
	SystemProcsBlocked      MetricID = "system/procs_blocked"
	SystemInterruptsTotal   MetricID = "system/interrupts_total"
	SystemCPUStat           MetricID = "system/cpu_stat"
	NetDevRxBytes           MetricID = "net/rx_bytes"
	NetDevRxPackets         MetricID = "net/rx_packets"
	NetDevRxErrors          MetricID = "net/rx_errors"
	NetDevRxDropped         MetricID = "net/rx_dropped"
	NetDevRxFifo            MetricID = "net/rx_fifo"
	NetDevRxFrame           MetricID = "net/rx_frame"
	NetDevRxCompressed      MetricID = "net/rx_compressed"
	NetDevRxMulticast       MetricID = "net/rx_multicast"
	NetDevTxBytes           MetricID = "net/tx_bytes"
	NetDevTxPackets         MetricID = "net/tx_packets"
	NetDevTxErrors          MetricID = "net/tx_errors"
	NetDevTxDropped         MetricID = "net/tx_dropped"
	NetDevTxFifo            MetricID = "net/tx_fifo"
	NetDevTxCollisions      MetricID = "net/tx_collisions"
	NetDevTxCarrier         MetricID = "net/tx_carrier"
	NetDevTxCompressed      MetricID = "net/tx_compressed"
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

// MetricInterface provides a common interface for all metric types
type MetricInterface interface {
	Record(labelValues map[string]string, value interface{}) error
}
