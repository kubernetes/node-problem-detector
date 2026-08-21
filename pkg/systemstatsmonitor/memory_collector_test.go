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
	"testing"

	"github.com/stretchr/testify/assert"
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

	mc.collect()
}
