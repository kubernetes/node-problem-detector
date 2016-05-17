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

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"

	"k8s.io/node-problem-detector/pkg/types"
)

// ConvertToAPICondition converts the internal node condition to api.NodeCondition.
func ConvertToAPICondition(condition types.Condition) api.NodeCondition {
	return api.NodeCondition{
		Type:               api.NodeConditionType(condition.Type),
		Status:             ConvertToAPIConditionStatus(condition.Status),
		LastTransitionTime: ConvertToAPITimestamp(condition.Transition),
		Reason:             condition.Reason,
		Message:            condition.Message,
	}
}

// ConvertToAPIConditionStatus converts the internal node condition status to api.ConditionStatus.
func ConvertToAPIConditionStatus(status bool) api.ConditionStatus {
	if status {
		return api.ConditionTrue
	}
	return api.ConditionFalse
}

// ConvertToAPIEventType converts the internal severity to event type.
func ConvertToAPIEventType(severity types.Severity) string {
	switch severity {
	case types.Info:
		return api.EventTypeNormal
	case types.Warn:
		return api.EventTypeWarning
	default:
		// Should never get here, just in case
		return api.EventTypeNormal
	}
}

// ConvertToAPITimestamp converts the timestamp to unversioned.Time
func ConvertToAPITimestamp(timestamp time.Time) unversioned.Time {
	return unversioned.NewTime(timestamp)
}
