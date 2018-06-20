/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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
	"k8s.io/node-problem-detector/pkg/types"
	"time"
)

type Status int

const (
	OK      Status = 0
	NonOK   Status = 1
	Unknown Status = 2
)

// Result is the custom plugin check result returned by plugin.
type Result struct {
	Rule       *CustomRule
	ExitStatus Status
	Message    string
}

// CustomRule describes how custom plugin monitor should invoke and analyze plugins.
type CustomRule struct {
	// Type is the type of the problem.
	Type types.Type `json:"type"`
	// Condition is the type of the condition the problem triggered. Notice that
	// the Condition field should be set only when the problem is permanent, or
	// else the field will be ignored.
	Condition string `json:"condition"`
	// Reason is the short reason of the problem.
	Reason string `json:"reason"`
	// Path is the path to the custom plugin.
	Path string `json:"path"`
	// Args is the args passed to the custom plugin.
	Args []string `json:"args"`
	// Timeout is the timeout string for the custom plugin to execute.
	TimeoutString *string `json:"timeout"`
	// Timeout is the timeout for the custom plugin to execute.
	Timeout *time.Duration `json:"-"`
	// TODO(andyxning) Add support for per-rule interval.
}
