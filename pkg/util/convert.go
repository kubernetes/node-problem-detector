/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package util

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/node-problem-detector/pkg/types"
)

// ConvertToAPICondition converts the internal node condition to v1.NodeCondition.
func ConvertToAPICondition(condition types.Condition) v1.NodeCondition {
	return v1.NodeCondition{
		Type:               v1.NodeConditionType(condition.Type),
		Status:             ConvertToAPIConditionStatus(condition.Status),
		LastTransitionTime: ConvertToAPITimestamp(condition.Transition),
		Reason:             condition.Reason,
		Message:            condition.Message,
	}
}

// ConvertToAPIConditionStatus converts the internal node condition status to v1.ConditionStatus.
func ConvertToAPIConditionStatus(status types.ConditionStatus) v1.ConditionStatus {
	switch status {
	case types.True:
		return v1.ConditionTrue
	case types.False:
		return v1.ConditionFalse
	case types.Unknown:
		return v1.ConditionUnknown
	default:
		panic("unknown condition status")
	}
}

// ConvertToAPIEventType converts the internal severity to event type.
func ConvertToAPIEventType(severity types.Severity) string {
	switch severity {
	case types.Info:
		return v1.EventTypeNormal
	case types.Warn:
		return v1.EventTypeWarning
	default:
		// Should never get here, just in case
		return v1.EventTypeNormal
	}
}

// ConvertToAPITimestamp converts the timestamp to metav1.Time
func ConvertToAPITimestamp(timestamp time.Time) metav1.Time {
	return metav1.NewTime(timestamp)
}
