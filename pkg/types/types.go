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

	"github.com/spf13/pflag"
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

// ConditionStatus is the status of the condition.
type ConditionStatus string

const (
	// True means the condition status is true.
	True ConditionStatus = "True"
	// False means the condition status is false.
	False ConditionStatus = "False"
	// Unknown means the condition status is unknown.
	Unknown ConditionStatus = "Unknown"
)

// Condition is the node condition used internally by problem detector.
type Condition struct {
	// Type is the condition type. It should describe the condition of node in problem. For example
	// KernelDeadlock, OutOfResource etc.
	Type string `json:"type"`
	// Status indicates whether the node is in the condition or not.
	Status ConditionStatus `json:"status"`
	// Transition is the time when the node transits to this condition.
	Transition time.Time `json:"transition"`
	// Reason is a short reason of why node goes into this condition.
	Reason string `json:"reason"`
	// Message is a human-readable message of why node goes into this condition.
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
	// Message is a human-readable message of why the event is generated.
	Message string `json:"message"`
}

// Status is the status other problem daemons should report to node problem detector.
type Status struct {
	// Source is the name of the problem daemon.
	Source string `json:"source"`
	// Events are temporary node problem events. If the status is only a condition update,
	// this field could be nil. Notice that the events should be sorted from oldest to newest.
	Events []Event `json:"events"`
	// Conditions are the permanent node conditions. The problem daemon should always report the
	// newest node conditions in this field.
	Conditions []Condition `json:"conditions"`
}

// Type is the type of the problem.
type Type string

const (
	// Temp means the problem is temporary, only need to report an event.
	Temp Type = "temporary"
	// Perm means the problem is permanent, need to change the node condition.
	Perm Type = "permanent"
)

// Monitor monitors the system and reports problems and metrics according to the rules.
type Monitor interface {
	// Start starts the monitor.
	// The Status channel is used to report problems. If the Monitor does not report any
	// problem (i.e. metrics reporting only), the channel should be set to nil.
	Start() (<-chan *Status, error)
	// Stop stops the monitor.
	Stop()
}

// Exporter exports machine health data to certain control plane.
type Exporter interface {
	// ExportProblems Export problems to the control plane.
	ExportProblems(*Status)
}

// ProblemDaemonType is the type of the problem daemon.
// One type of problem daemon may be used to initialize multiple problem daemon instances.
type ProblemDaemonType string

// ProblemDaemonConfigPathMap represents configurations on all types of problem daemons:
// 1) Each key represents a type of problem daemon.
// 2) Each value represents the config file paths to that type of problem daemon.
type ProblemDaemonConfigPathMap map[ProblemDaemonType]*[]string

// ProblemDaemonHandler represents the initialization handler for a type problem daemon.
type ProblemDaemonHandler struct {
	// CreateProblemDaemonOrDie initializes a problem daemon, panic if error occurs.
	CreateProblemDaemonOrDie func(string) Monitor
	// CmdOptionDescription explains how to configure the problem daemon from command line arguments.
	CmdOptionDescription string
}

// ExporterType is the type of the exporter.
type ExporterType string

// ExporterHandler represents the initialization handler for a type of exporter.
type ExporterHandler struct {
	// CreateExporterOrDie initializes an exporter, panic if error occurs.
	CreateExporterOrDie func(CommandLineOptions) Exporter
	// CmdOptionDescription explains how to configure the exporter from command line arguments.
	Options CommandLineOptions
}

type CommandLineOptions interface {
	SetFlags(*pflag.FlagSet)
}
