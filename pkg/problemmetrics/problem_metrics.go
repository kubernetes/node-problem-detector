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
	"errors"
	"fmt"
	"sync"

	"github.com/golang/glog"

	"k8s.io/node-problem-detector/pkg/util/metrics"
)

// GlobalProblemMetricsManager is a singleton of ProblemMetricsManager,
// which should be used to manage all problem-converted metrics across all
// problem daemons.
var GlobalProblemMetricsManager *ProblemMetricsManager

func init() {
	GlobalProblemMetricsManager = NewProblemMetricsManagerOrDie()
}

// ProblemMetricsManager manages problem-converted metrics.
// ProblemMetricsManager is thread-safe.
type ProblemMetricsManager struct {
	problemCounter           metrics.Int64MetricInterface
	problemGauge             metrics.Int64MetricInterface
	problemTypeToReason      map[string]string
	problemTypeToReasonMutex sync.Mutex
}

func NewProblemMetricsManagerOrDie() *ProblemMetricsManager {
	pmm := ProblemMetricsManager{}

	var err error
	pmm.problemCounter, err = metrics.NewInt64Metric(
		metrics.ProblemCounterID,
		string(metrics.ProblemCounterID),
		"Number of times a specific type of problem have occurred.",
		"1",
		metrics.Sum,
		[]string{"reason"})
	if err != nil {
		glog.Fatalf("Failed to create problem_counter metric: %v", err)
	}

	pmm.problemGauge, err = metrics.NewInt64Metric(
		metrics.ProblemGaugeID,
		string(metrics.ProblemGaugeID),
		"Whether a specific type of problem is affecting the node or not.",
		"1",
		metrics.LastValue,
		[]string{"type", "reason"})
	if err != nil {
		glog.Fatalf("Failed to create problem_gauge metric: %v", err)
	}

	pmm.problemTypeToReason = make(map[string]string)

	return &pmm
}

// IncrementProblemCounter increments the value of a problem counter.
func (pmm *ProblemMetricsManager) IncrementProblemCounter(reason string, count int64) error {
	if pmm.problemCounter == nil {
		return errors.New("problem counter is being incremented before initialized.")
	}

	return pmm.problemCounter.Record(map[string]string{"reason": reason}, count)
}

// SetProblemGauge sets the value of a problem gauge.
func (pmm *ProblemMetricsManager) SetProblemGauge(problemType string, reason string, value bool) error {
	if pmm.problemGauge == nil {
		return errors.New("problem gauge is being set before initialized.")
	}

	pmm.problemTypeToReasonMutex.Lock()
	defer pmm.problemTypeToReasonMutex.Unlock()

	// We clear the last reason, because the expected behavior is that at any point of time,
	// for each type of permanent problem, there should be at most one reason got set to 1.
	// This behavior is consistent with the behavior of node condition in Kubernetes.
	// However, problemGauges with different "type" and "reason" are considered as different
	// metrics in Prometheus. So we need to clear the previous metrics explicitly.
	if lastReason, ok := pmm.problemTypeToReason[problemType]; ok {
		err := pmm.problemGauge.Record(map[string]string{"type": problemType, "reason": lastReason}, 0)
		if err != nil {
			return fmt.Errorf("failed to clear previous reason %q for type %q: %v",
				problemType, lastReason, err)
		}
	}

	pmm.problemTypeToReason[problemType] = reason

	var valueInt int64
	if value {
		valueInt = 1
	}
	return pmm.problemGauge.Record(map[string]string{"type": problemType, "reason": reason}, valueInt)
}
