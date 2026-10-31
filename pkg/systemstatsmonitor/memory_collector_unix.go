//go:build unix

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
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/procfs"
	"k8s.io/klog/v2"
)

var (
	zswapMaxPoolPercentPath    = "/sys/module/zswap/parameters/max_pool_percent"
	procVMStatPath             = "/proc/vmstat"
	zswapWrittenBackPagesDebug = "/sys/kernel/debug/zswap/written_back_pages"
)

func (mc *memoryCollector) collect() {
	if mc == nil {
		return
	}

	proc, err := procfs.NewDefaultFS()
	if err != nil {
		klog.Errorf("Failed to find /proc mount point: %v", err)
		return
	}
	meminfo, err := proc.Meminfo()
	if err != nil {
		klog.Errorf("Failed to retrieve memory stats: %v", err)
		return
	}

	if mc.mBytesUsed != nil {
		if meminfo.MemFree != nil {
			if err := mc.mBytesUsed.Record(map[string]string{stateLabel: "free"}, int64(*meminfo.MemFree)*1024); err != nil {
				klog.Errorf("Failed to record memory bytes used for free state: %v", err)
			}
		}
		if meminfo.Buffers != nil {
			if err := mc.mBytesUsed.Record(map[string]string{stateLabel: "buffered"}, int64(*meminfo.Buffers)*1024); err != nil {
				klog.Errorf("Failed to record memory bytes used for buffered state: %v", err)
			}
		}
		if meminfo.Cached != nil {
			if err := mc.mBytesUsed.Record(map[string]string{stateLabel: "cached"}, int64(*meminfo.Cached)*1024); err != nil {
				klog.Errorf("Failed to record memory bytes used for cached state: %v", err)
			}
		}
		if meminfo.Slab != nil {
			if err := mc.mBytesUsed.Record(map[string]string{stateLabel: "slab"}, int64(*meminfo.Slab)*1024); err != nil {
				klog.Errorf("Failed to record memory bytes used for slab state: %v", err)
			}
		}
		if meminfo.MemTotal != nil && meminfo.MemFree != nil && meminfo.Buffers != nil && meminfo.Cached != nil && meminfo.Slab != nil {
			memUsed := *meminfo.MemTotal - *meminfo.MemFree - *meminfo.Buffers - *meminfo.Cached - *meminfo.Slab
			if err := mc.mBytesUsed.Record(map[string]string{stateLabel: "used"}, int64(memUsed)*1024); err != nil {
				klog.Errorf("Failed to record memory bytes used for used state: %v", err)
			}
		}
	}

	if mc.mPercentUsed != nil && meminfo.MemTotal != nil && *meminfo.MemTotal > 0 &&
		meminfo.MemFree != nil && meminfo.Buffers != nil && meminfo.Cached != nil && meminfo.Slab != nil {
		ratio := float64(*meminfo.MemTotal-*meminfo.MemFree-*meminfo.Buffers-*meminfo.Cached-*meminfo.Slab) / float64(*meminfo.MemTotal)
		if err := mc.mPercentUsed.Record(map[string]string{stateLabel: "used"}, float64(ratio*100.0)); err != nil {
			klog.Errorf("Failed to record memory percent used: %v", err)
		}
	}

	if mc.mDirtyUsed != nil {
		if meminfo.Dirty != nil {
			if err := mc.mDirtyUsed.Record(map[string]string{stateLabel: "dirty"}, int64(*meminfo.Dirty)*1024); err != nil {
				klog.Errorf("Failed to record dirty used memory: %v", err)
			}
		}
		if meminfo.Writeback != nil {
			if err := mc.mDirtyUsed.Record(map[string]string{stateLabel: "writeback"}, int64(*meminfo.Writeback)*1024); err != nil {
				klog.Errorf("Failed to record writeback used memory: %v", err)
			}
		}
	}

	if mc.mAnonymousUsed != nil {
		if meminfo.ActiveAnon != nil {
			if err := mc.mAnonymousUsed.Record(map[string]string{stateLabel: "active"}, int64(*meminfo.ActiveAnon)*1024); err != nil {
				klog.Errorf("Failed to record active anonymous used memory: %v", err)
			}
		}
		if meminfo.InactiveAnon != nil {
			if err := mc.mAnonymousUsed.Record(map[string]string{stateLabel: "inactive"}, int64(*meminfo.InactiveAnon)*1024); err != nil {
				klog.Errorf("Failed to record inactive anonymous used memory: %v", err)
			}
		}
	}

	if mc.mPageCacheUsed != nil {
		if meminfo.ActiveFile != nil {
			if err := mc.mPageCacheUsed.Record(map[string]string{stateLabel: "active"}, int64(*meminfo.ActiveFile)*1024); err != nil {
				klog.Errorf("Failed to record active page cache used memory: %v", err)
			}
		}
		if meminfo.InactiveFile != nil {
			if err := mc.mPageCacheUsed.Record(map[string]string{stateLabel: "inactive"}, int64(*meminfo.InactiveFile)*1024); err != nil {
				klog.Errorf("Failed to record inactive page cache used memory: %v", err)
			}
		}
	}

	if mc.mUnevictableUsed != nil && meminfo.Unevictable != nil {
		if err := mc.mUnevictableUsed.Record(map[string]string{}, int64(*meminfo.Unevictable)*1024); err != nil {
			klog.Errorf("Failed to record unevictable used memory: %v", err)
		}
	}

	mc.collectZswap(&meminfo)
}

func (mc *memoryCollector) collectZswap(meminfo *procfs.Meminfo) {
	if mc.mZswapBytesUsed != nil {
		var bytesUsed int64
		if meminfo.Zswap != nil && *meminfo.Zswap > 0 {
			bytesUsed = int64(*meminfo.Zswap) * 1024 // kB -> B
		}
		if err := mc.mZswapBytesUsed.Record(map[string]string{}, bytesUsed); err != nil {
			klog.Errorf("Failed to record zswap bytes used: %v", err)
		}
	}

	if mc.mZswapCompressionEfficiency != nil {
		var efficiency float64
		if meminfo.Zswap != nil && *meminfo.Zswap > 0 && meminfo.Zswapped != nil {
			efficiency = float64(*meminfo.Zswapped) / float64(*meminfo.Zswap)
		}
		if err := mc.mZswapCompressionEfficiency.Record(map[string]string{}, efficiency); err != nil {
			klog.Errorf("Failed to record zswap compression efficiency: %v", err)
		}
	}

	if mc.mZswapPoolPercent != nil {
		poolPercent := calculateZswapPoolPercent(meminfo, zswapMaxPoolPercentPath)
		if err := mc.mZswapPoolPercent.Record(map[string]string{}, poolPercent); err != nil {
			klog.Errorf("Failed to record zswap pool percent: %v", err)
		}
	}

	if mc.mZswapWritebackPages != nil {
		writebackPages := readZswapWritebackPages(procVMStatPath, zswapWrittenBackPagesDebug)
		if err := mc.mZswapWritebackPages.Record(map[string]string{}, writebackPages); err != nil {
			klog.Errorf("Failed to record zswap writeback pages: %v", err)
		}
	}
}

func calculateZswapPoolPercent(meminfo *procfs.Meminfo, maxPoolPctPath string) float64 {
	if meminfo == nil || meminfo.Zswap == nil || *meminfo.Zswap == 0 || meminfo.MemTotal == nil || *meminfo.MemTotal == 0 {
		return 0
	}
	data, err := os.ReadFile(maxPoolPctPath)
	if err != nil {
		return 0
	}
	maxPoolPct, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil || maxPoolPct <= 0 {
		return 0
	}
	maxPoolKB := float64(*meminfo.MemTotal) * (maxPoolPct / 100.0)
	if maxPoolKB <= 0 {
		return 0
	}
	return (float64(*meminfo.Zswap) / maxPoolKB) * 100.0
}

func readZswapWritebackPages(vmstatPath, debugfsPath string) int64 {
	if data, err := os.ReadFile(vmstatPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "zswpwb" {
				if val, err := strconv.ParseInt(fields[1], 10, 64); err == nil && val >= 0 {
					return val
				}
				break
			}
		}
	}
	if data, err := os.ReadFile(debugfsPath); err == nil {
		if val, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil && val >= 0 {
			return val
		}
	}
	return 0
}
