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
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/util/metrics"
)

func TestNewProblem(t *testing.T) {
	testCases := []struct {
		name            string
		reasons         []string
		counts          []int64
		expectedMetrics []metrics.Int64MetricRepresentation
	}{
		{
			name:            "no problem at all",
			reasons:         []string{},
			counts:          []int64{},
			expectedMetrics: []metrics.Int64MetricRepresentation{},
		},
		{
			name:    "one problem happened",
			reasons: []string{"foo"},
			counts:  []int64{1},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "foo"},
					Value:  1,
				},
			},
		},
		{
			name:    "one problem happened twice",
			reasons: []string{"foo", "foo"},
			counts:  []int64{1, 1},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "foo"},
					Value:  2,
				},
			},
		},
		{
			name:    "two problem happened various times",
			reasons: []string{"foo", "bar", "foo"},
			counts:  []int64{1, 1, 1},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "foo"},
					Value:  2,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "bar"},
					Value:  1,
				},
			},
		},
		{
			name:    "two problem initialized",
			reasons: []string{"foo", "bar"},
			counts:  []int64{0, 0},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "foo"},
					Value:  0,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "bar"},
					Value:  0,
				},
			},
		},
		{
			name:    "two problem first initialized, then happened various times",
			reasons: []string{"foo", "bar", "foo", "bar", "foo"},
			counts:  []int64{0, 0, 1, 1, 1},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "foo"},
					Value:  2,
				},
				{
					Name:   "problem_counter",
					Labels: map[string]string{"reason": "bar"},
					Value:  1,
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			pmm, fakeProblemCounter, fakeProblemGauge := NewProblemMetricsManagerStub()

			for idx, reason := range test.reasons {
				pmm.IncrementProblemCounter(reason, test.counts[idx])
			}

			gotMetrics := append(fakeProblemCounter.ListMetrics(), fakeProblemGauge.ListMetrics()...)
			assert.ElementsMatch(t, test.expectedMetrics, gotMetrics,
				"expected metrics: %+v, got: %+v", test.expectedMetrics, gotMetrics)
		})
	}
}

func TestSetProblemGauge(t *testing.T) {
	type argumentType struct {
		problemType string
		reason      string
		value       bool
	}

	testCases := []struct {
		name            string
		arguments       []argumentType
		expectedMetrics []metrics.Int64MetricRepresentation
	}{
		{
			name:            "no permanent problem at all",
			arguments:       []argumentType{},
			expectedMetrics: []metrics.Int64MetricRepresentation{},
		},
		{
			name: "one permanent problem was set once",
			arguments: []argumentType{
				{"ProblemTypeA", "ReasonFoo", true},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": "ReasonFoo"},
					Value:  1,
				},
			},
		},
		{
			name: "one permanent problem was set twice with same reason",
			arguments: []argumentType{
				{"ProblemTypeA", "ReasonFoo", true},
				{"ProblemTypeA", "ReasonFoo", true},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": "ReasonFoo"},
					Value:  1,
				},
			},
		},
		{
			name: "one permanent problem was set twice with different reasons",
			arguments: []argumentType{
				{"ProblemTypeA", "ReasonFoo", true},
				{"ProblemTypeA", "ReasonBar", true},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": "ReasonFoo"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": "ReasonBar"},
					Value:  1,
				},
			},
		},
		{
			name: "one permanent problem was set then cleared",
			arguments: []argumentType{
				{"ProblemTypeA", "ReasonFoo", true},
				{"ProblemTypeA", "", false},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": ""},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": "ReasonFoo"},
					Value:  0,
				},
			},
		},
		{
			name: "one permanent problem was set, cleared, and set again",
			arguments: []argumentType{
				{"ProblemTypeA", "ReasonFoo", true},
				{"ProblemTypeA", "", false},
				{"ProblemTypeA", "ReasonBar", true},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": ""},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": "ReasonFoo"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": "ReasonBar"},
					Value:  1,
				},
			},
		},
		{
			name: "two permanent problems were set and one of them got cleared",
			arguments: []argumentType{
				{"ProblemTypeA", "ReasonFoo", true},
				{"ProblemTypeB", "ReasonBar", true},
				{"ProblemTypeA", "", false},
			},
			expectedMetrics: []metrics.Int64MetricRepresentation{
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": ""},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeA", "reason": "ReasonFoo"},
					Value:  0,
				},
				{
					Name:   "problem_gauge",
					Labels: map[string]string{"type": "ProblemTypeB", "reason": "ReasonBar"},
					Value:  1,
				},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			pmm, fakeProblemCounter, fakeProblemGauge := NewProblemMetricsManagerStub()

			for _, argument := range test.arguments {
				pmm.SetProblemGauge(argument.problemType, argument.reason, argument.value)
			}

			gotMetrics := append(fakeProblemCounter.ListMetrics(), fakeProblemGauge.ListMetrics()...)
			assert.ElementsMatch(t, test.expectedMetrics, gotMetrics,
				"expected metrics: %+v, got: %+v", test.expectedMetrics, gotMetrics)
		})
	}
}
