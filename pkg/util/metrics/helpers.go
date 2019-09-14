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
	"fmt"
	"strings"
	"sync"

	pcm "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"go.opencensus.io/tag"
)

var tagMap map[string]tag.Key
var tagMapMutex sync.RWMutex

func init() {
	tagMapMutex.Lock()
	tagMap = make(map[string]tag.Key)
	tagMapMutex.Unlock()
}

// Aggregation defines how measurements should be aggregated into data points.
type Aggregation string

const (
	// LastValue means last measurement overwrites previous measurements (gauge metric).
	LastValue Aggregation = "LastValue"
	// Sum means last measurement will be added onto previous measurements (counter metric).
	Sum Aggregation = "Sum"
)

func getTagKeysFromNames(tagNames []string) ([]tag.Key, error) {
	tagMapMutex.Lock()
	defer tagMapMutex.Unlock()

	var tagKeys []tag.Key
	var err error
	for _, tagName := range tagNames {
		tagKey, ok := tagMap[tagName]
		if !ok {
			tagKey, err = tag.NewKey(tagName)
			if err != nil {
				return []tag.Key{}, fmt.Errorf("failed to create tag %q: %v", tagName, err)
			}
			tagMap[tagName] = tagKey
		}
		tagKeys = append(tagKeys, tagKey)
	}
	return tagKeys, nil
}

// ParsePrometheusMetrics parses Prometheus formatted metrics into metrics under Float64MetricRepresentation.
//
// Note: Prometheus's go library stores all counter/gauge-typed metric values under float64.
func ParsePrometheusMetrics(metricsText string) ([]Float64MetricRepresentation, error) {
	var metrics []Float64MetricRepresentation

	var textParser expfmt.TextParser
	metricFamilies, err := textParser.TextToMetricFamilies(strings.NewReader(metricsText))
	if err != nil {
		return metrics, err
	}

	for _, metricFamily := range metricFamilies {
		for _, metric := range metricFamily.Metric {
			labels := make(map[string]string)
			for _, labelPair := range metric.Label {
				labels[*labelPair.Name] = *labelPair.Value
			}

			var value float64
			if *metricFamily.Type == pcm.MetricType_COUNTER {
				value = *metric.Counter.Value
			} else if *metricFamily.Type == pcm.MetricType_GAUGE {
				value = *metric.Gauge.Value
			} else {
				return metrics, fmt.Errorf("unexpected MetricType %s for metric %s",
					pcm.MetricType_name[int32(*metricFamily.Type)], *metricFamily.Name)
			}

			metrics = append(metrics, Float64MetricRepresentation{*metricFamily.Name, labels, value})
		}
	}

	return metrics, nil
}

// GetFloat64Metric finds the metric matching provided name and labels.
// When strictLabelMatching is set to true, the founded metric labels are identical to the provided labels;
// when strictLabelMatching is set to false, the founded metric labels are a superset of the provided labels.
func GetFloat64Metric(metrics []Float64MetricRepresentation, name string, labels map[string]string,
	strictLabelMatching bool) (Float64MetricRepresentation, error) {
	for _, metric := range metrics {
		if metric.Name != name {
			continue
		}
		if strictLabelMatching && len(metric.Labels) != len(labels) {
			continue
		}
		sameLabels := true
		for key, value := range labels {
			if metric.Labels[key] != value {
				sameLabels = false
				break
			}
		}
		if !sameLabels {
			continue
		}
		return metric, nil
	}
	return Float64MetricRepresentation{}, fmt.Errorf("no matching metric found")
}
