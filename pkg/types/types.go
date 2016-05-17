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

package types

import (
	"time"
)

// The following types are used internally in problem detector. In the future this could be the
// interface between node problem detector and other problem daemons.
// We added these types because:
// 1) The kubernetes api packages are too heavy.
// 2) We want to make the interface independent with kubernetes api change.

// Severity is the severity of the problem event. Now we only have 2 severity levels: Info and Warn,
// which are corresponding to the current kubernetes event types. We may want to add more severity
// levels in the future.
type Severity string

const (
	// Info is translated to a normal event.
	Info Severity = "info"
	// Warn is translated to a warning event.
	Warn Severity = "warn"
)

// Condition is the node condition used internally by problem detector.
type Condition struct {
	// Type is the condition type. It should describe the condition of node in problem. For example
	// KernelDeadlock, OutOfResource etc.
	Type string `json:"type"`
	// Status indicates whether the node is in the condition or not.
	Status bool `json:"status"`
	// Transition is the time when the node transits to this condition.
	Transition time.Time `json:"transition"`
	// Reason is a short reason of why node goes into this condition.
	Reason string `json:"reason"`
	// Message is a human readable message of why node goes into this condition.
	Message string `json:"message"`
}

// Event is the event used internally by node problem detector.
type Event struct {
	// Severity is the severity level of the event.
	Severity Severity `json:"severity"`
	// Timestamp is the time when the event is generated.
	Timestamp time.Time `json:"timestamp"`
	// Reason is a short reason of why the event is generated.
	Reason string `json:"reason"`
	// Message is a human readable message of why the event is generated.
	Message string `json:"message"`
}

// Status is the status other problem daemons should report to node problem detector.
type Status struct {
	// Source is the name of the problem daemon.
	Source string `json:"source"`
	// Event is the temporary node problem event. If the status is only a condition update,
	// this field could be nil.
	Event *Event `json:"event"`
	// Condition is the permanent node condition. The problem daemon should always report the
	// newest node condition in this field.
	Condition Condition `json:"condition"`
}
