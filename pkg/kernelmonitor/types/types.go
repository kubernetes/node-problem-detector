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

// KernelLog is the log item returned by translator. It's very easy to extend this
// to support other log monitoring, such as docker log monitoring.
type KernelLog struct {
	Timestamp int64 // microseconds since kernel boot
	Message   string
}

// Type is the type of the kernel problem.
type Type string

const (
	// Temp means the kernel problem is temporary, only need to report an event.
	Temp Type = "temporary"
	// Perm means the kernel problem is permanent, need to change the node condition.
	Perm Type = "permanent"
)

// Rule describes how kernel monitor should analyze the kernel log.
type Rule struct {
	// Type is the type of matched kernel problem.
	Type Type `json:"type"`
	// Reason is the short reason of the kernel problem.
	Reason string `json:"reason"`
	// Pattern is the regular expression to match the kernel problem in kernel log.
	// Notice that the pattern must match to the end of the line.
	Pattern string `json:"pattern"`
}
