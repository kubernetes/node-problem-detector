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
	"github.com/golang/glog"
	ssmtypes "k8s.io/node-problem-detector/pkg/systemstatsmonitor/types"
	"k8s.io/node-problem-detector/pkg/util/metrics"
	"k8s.io/node-problem-detector/pkg/util/metrics/system"
)

type fileHandlerCollector struct {
	mFileHandleCount *metrics.Int64Metric

	config *ssmtypes.FileHandlerStatsConfig
}

func NewFileHandlerCollectorOrDie(fileHandlerConfig *ssmtypes.FileHandlerStatsConfig) *fileHandlerCollector {
	fhc := fileHandlerCollector{config: fileHandlerConfig}
	var err error

	fhc.mFileHandleCount, err = metrics.NewInt64Metric(
		metrics.FileHandleCountId,
		fileHandlerConfig.MetricsConfigs[string(metrics.FileHandleCountId)].DisplayName,
		"The total number of the file handles currently used.",
		"1",
		metrics.LastValue,
		[]string{})

	if err != nil {
		glog.Fatalf("Error initializing metric for %q: %v", metrics.FileHandleCountId, err)
	}
	return &fhc
}

func (fhc *fileHandlerCollector) collect() {
	if fhc.mFileHandleCount == nil {
		return
	}
	fileHandleCount, err := system.GetFileHandleCurrentlyInUseCount()
	if err != nil {
		glog.Errorf("Failed to retrieve File Handle Count: %v", err)
		return
	}
	fhc.mFileHandleCount.Record(map[string]string{}, fileHandleCount)
}
