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

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

type memoryCollector struct {
	mBytesUsed       *metrics.Int64Metric
	mAnonymousUsed   *metrics.Int64Metric
	mPageCacheUsed   *metrics.Int64Metric
	mUnevictableUsed *metrics.Int64Metric
	mDirtyUsed       *metrics.Int64Metric

	config *ssmtypes.MemoryStatsConfig
}

func NewMemoryCollectorOrDie(memoryConfig *ssmtypes.MemoryStatsConfig) *memoryCollector {
	mc := memoryCollector{config: memoryConfig}

	var err error

	mc.mBytesUsed, err = metrics.NewInt64Metric(
		metrics.MemoryBytesUsedID,
		memoryConfig.MetricsConfigs[string(metrics.MemoryBytesUsedID)].DisplayName,
		"Memory usage by each memory state, in Bytes. Summing values of all states yields the total memory on the node.",
		"Byte",
		metrics.LastValue,
		[]string{stateLabel})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.MemoryBytesUsedID, err)
	}

	mc.mAnonymousUsed, err = metrics.NewInt64Metric(
		metrics.MemoryAnonymousUsedID,
		memoryConfig.MetricsConfigs[string(metrics.MemoryAnonymousUsedID)].DisplayName,
		"Anonymous memory usage, in Bytes. Summing values of all states yields the total anonymous memory used.",
		"Byte",
		metrics.LastValue,
		[]string{stateLabel})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.MemoryAnonymousUsedID, err)
	}

	mc.mPageCacheUsed, err = metrics.NewInt64Metric(
		metrics.MemoryPageCacheUsedID,
		memoryConfig.MetricsConfigs[string(metrics.MemoryPageCacheUsedID)].DisplayName,
		"Page cache memory usage, in Bytes. Summing values of all states yields the total anonymous memory used.",
		"Byte",
		metrics.LastValue,
		[]string{stateLabel})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.MemoryPageCacheUsedID, err)
	}

	mc.mUnevictableUsed, err = metrics.NewInt64Metric(
		metrics.MemoryUnevictableUsedID,
		memoryConfig.MetricsConfigs[string(metrics.MemoryUnevictableUsedID)].DisplayName,
		"Unevictable memory usage, in Bytes",
		"Byte",
		metrics.LastValue,
		[]string{})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.MemoryUnevictableUsedID, err)
	}

	mc.mDirtyUsed, err = metrics.NewInt64Metric(
		metrics.MemoryDirtyUsedID,
		memoryConfig.MetricsConfigs[string(metrics.MemoryDirtyUsedID)].DisplayName,
		"Dirty pages usage, in Bytes. Dirty means the memory is waiting to be written back to disk, and writeback means the memory is actively being written back to disk.",
		"Byte",
		metrics.LastValue,
		[]string{stateLabel})
	if err != nil {
		klog.Fatalf("Error initializing metric for %q: %v", metrics.MemoryDirtyUsedID, err)
	}

	return &mc
}
