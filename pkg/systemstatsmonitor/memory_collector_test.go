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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/procfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
)

const (
	fakeZswapConfig = `
{
	"metricsConfigs": {
		"memory/zswap_bytes_used": {
			"displayName": "memory/zswap_bytes_used"
		},
		"memory/zswap_compression_efficiency": {
			"displayName": "memory/zswap_compression_efficiency"
		},
		"memory/zswap_pool_percent": {
			"displayName": "memory/zswap_pool_percent"
		},
		"memory/zswap_writeback_pages": {
			"displayName": "memory/zswap_writeback_pages"
		}
	}
}
`
)

func TestMemoryCollector(t *testing.T) {
	// Original test
	mc := NewMemoryCollectorOrDie(&ssmtypes.MemoryStatsConfig{})
	mc.collect()

	// Ensure zswap metrics are nil when not in config
	assert.Nil(t, mc.mZswapBytesUsed)
	assert.Nil(t, mc.mZswapCompressionEfficiency)
	assert.Nil(t, mc.mZswapPoolPercent)
	assert.Nil(t, mc.mZswapWritebackPages)
}

func TestMemoryCollectorZswap(t *testing.T) {
	cfg := &ssmtypes.MemoryStatsConfig{}
	if err := json.Unmarshal([]byte(fakeZswapConfig), cfg); err != nil {
		t.Fatalf("cannot load memory config: %s", err)
	}

	mc := NewMemoryCollectorOrDie(cfg)

	// Ensure zswap metrics are initialized when in config
	assert.NotNil(t, mc.mZswapBytesUsed)
	assert.NotNil(t, mc.mZswapCompressionEfficiency)
	assert.NotNil(t, mc.mZswapPoolPercent)
	assert.NotNil(t, mc.mZswapWritebackPages)

	mc.collect()
}

func TestCalculateZswapPoolPercent(t *testing.T) {
	tmpDir := t.TempDir()
	maxPoolPath := filepath.Join(tmpDir, "max_pool_percent")
	require.NoError(t, os.WriteFile(maxPoolPath, []byte("20\n"), 0o600))

	memTotal := uint64(1000000) // 1,000,000 kB -> max pool (20%) = 200,000 kB
	zswap := uint64(50000)      // 50,000 kB -> 25% of max pool
	meminfo := &procfs.Meminfo{
		MemTotal: &memTotal,
		Zswap:    &zswap,
	}

	assert.InDelta(t, 25.0, calculateZswapPoolPercent(meminfo, maxPoolPath), 1e-6)
	assert.Equal(t, 0.0, calculateZswapPoolPercent(nil, maxPoolPath))
	assert.Equal(t, 0.0, calculateZswapPoolPercent(meminfo, filepath.Join(tmpDir, "nonexistent")))
}

func TestReadZswapWritebackPages(t *testing.T) {
	tmpDir := t.TempDir()
	vmstatPath := filepath.Join(tmpDir, "vmstat")
	debugPath := filepath.Join(tmpDir, "written_back_pages")

	require.NoError(t, os.WriteFile(vmstatPath, []byte("pswpin 10\nzswpin 100\nzswpout 200\nzswpwb 42\n"), 0o600))
	require.NoError(t, os.WriteFile(debugPath, []byte("99\n"), 0o600))

	// Prefers /proc/vmstat zswpwb when available
	assert.Equal(t, int64(42), readZswapWritebackPages(vmstatPath, debugPath))

	// Falls back to debugfs when /proc/vmstat lacks zswpwb
	vmstatNoZswap := filepath.Join(tmpDir, "vmstat_no_zswap")
	require.NoError(t, os.WriteFile(vmstatNoZswap, []byte("pswpin 10\n"), 0o600))
	assert.Equal(t, int64(99), readZswapWritebackPages(vmstatNoZswap, debugPath))

	// Returns 0 when neither is available
	assert.Equal(t, int64(0), readZswapWritebackPages(vmstatNoZswap, filepath.Join(tmpDir, "missing")))
}
