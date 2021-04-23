/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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
	"testing"

	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
)

func TestHostCollector(t *testing.T) {
	hc := NewHostCollectorOrDie(&ssmtypes.HostStatsConfig{})
	hc.collect()
	val, ok := hc.tags["os_version"]
	if !ok {
		t.Errorf("tags[os_version] should exist.")
	} else if val == "" {
		t.Errorf("tags[os_version] should not be empty")
	}

	val, ok = hc.tags["kernel_version"]
	if !ok {
		t.Errorf("tags[kernel_version] should exist.")
	} else if val == "" {
		t.Errorf("tags[kernel_version] should not be empty")
	}
}
