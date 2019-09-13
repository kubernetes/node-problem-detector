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

package problemdaemon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/types"
)

func TestRegistration(t *testing.T) {
	fooMonitorFactory := func(types.CommandLineOptions) []types.Monitor {
		return []types.Monitor{}
	}
	fooMonitorHandler := types.ProblemDaemonHandler{
		CreateProblemDaemonOrDie: fooMonitorFactory,
		Options:                  nil,
	}

	barMonitorFactory := func(types.CommandLineOptions) []types.Monitor {
		return []types.Monitor{}
	}
	barMonitorHandler := types.ProblemDaemonHandler{
		CreateProblemDaemonOrDie: barMonitorFactory,
		Options:                  nil,
	}

	Register("foo", fooMonitorHandler)
	Register("bar", barMonitorHandler)

	expectedProblemDaemonNames := []types.ProblemDaemonType{"foo", "bar"}
	problemDaemonNames := GetProblemDaemonNames()
	assert.ElementsMatch(t, expectedProblemDaemonNames, problemDaemonNames)

	handlers = make(map[types.ProblemDaemonType]types.ProblemDaemonHandler)
}

func TestGetProblemDaemonHandlerOrDie(t *testing.T) {
	fooMonitorFactory := func(types.CommandLineOptions) []types.Monitor {
		return []types.Monitor{}
	}
	fooMonitorHandler := types.ProblemDaemonHandler{
		CreateProblemDaemonOrDie: fooMonitorFactory,
		Options:                  nil,
	}

	Register("foo", fooMonitorHandler)

	assert.NotPanics(t, func() { GetProblemDaemonHandlerOrDie("foo") })
	assert.Panics(t, func() { GetProblemDaemonHandlerOrDie("bar") })

	handlers = make(map[types.ProblemDaemonType]types.ProblemDaemonHandler)
}
