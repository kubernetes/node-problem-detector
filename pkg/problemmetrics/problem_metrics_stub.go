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

package problemmetrics

import (
	"k8s.io/node-problem-detector/pkg/util/metrics"
)

// NewProblemMetricsManagerStub creates a ProblemMetricsManager stubbed by fake metrics.
// The stubbed ProblemMetricsManager and fake metrics are returned.
func NewProblemMetricsManagerStub() (*ProblemMetricsManager, *metrics.FakeInt64Metric, *metrics.FakeInt64Metric) {
	fakeProblemCounter := metrics.NewFakeInt64Metric("problem_counter", metrics.Sum, []string{"reason"})
	fakeProblemGauge := metrics.NewFakeInt64Metric("problem_gauge", metrics.LastValue, []string{"type", "reason"})

	pmm := ProblemMetricsManager{}
	pmm.problemCounter = metrics.Int64MetricInterface(fakeProblemCounter)
	pmm.problemGauge = metrics.Int64MetricInterface(fakeProblemGauge)
	pmm.problemTypeToReason = make(map[string]string)

	return &pmm, fakeProblemCounter, fakeProblemGauge
}
