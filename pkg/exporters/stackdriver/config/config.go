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

package config

import (
	"time"

	"k8s.io/node-problem-detector/pkg/exporters/stackdriver/gce"
)

var (
	defaultExportPeriod          = (60 * time.Second).String()
	defaultEndpoint              = "monitoring.googleapis.com:443"
	defaultMetadataFetchTimeout  = (600 * time.Second).String()
	defaultMetadataFetchInterval = (10 * time.Second).String()
)

type StackdriverExporterConfig struct {
	ExportPeriod                string       `json:"exportPeriod"`
	APIEndpoint                 string       `json:"apiEndpoint"`
	GCEMetadata                 gce.Metadata `json:"gceMetadata"`
	MetadataFetchTimeout        string       `json:"metadataFetchTimeout"`
	MetadataFetchInterval       string       `json:"metadataFetchInterval"`
	PanicOnMetadataFetchFailure bool         `json:"panicOnMetadataFetchFailure"`
	CustomMetricPrefix          string       `json:"customMetricPrefix"`
}

// ApplyConfiguration applies default configurations.
func (sec *StackdriverExporterConfig) ApplyConfiguration() {
	if sec.ExportPeriod == "" {
		sec.ExportPeriod = defaultExportPeriod
	}
	if sec.MetadataFetchTimeout == "" {
		sec.MetadataFetchTimeout = defaultMetadataFetchTimeout
	}
	if sec.MetadataFetchInterval == "" {
		sec.MetadataFetchInterval = defaultMetadataFetchInterval
	}
	if sec.APIEndpoint == "" {
		sec.APIEndpoint = defaultEndpoint
	}
}
