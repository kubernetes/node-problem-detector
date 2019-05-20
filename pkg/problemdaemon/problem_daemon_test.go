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
	fooMonitorFactory := func(configPath string) types.Monitor {
		return nil
	}
	fooMonitorHandler := types.ProblemDaemonHandler{
		CreateProblemDaemonOrDie: fooMonitorFactory,
		CmdOptionDescription:     "foo option",
	}

	barMonitorFactory := func(configPath string) types.Monitor {
		return nil
	}
	barMonitorHandler := types.ProblemDaemonHandler{
		CreateProblemDaemonOrDie: barMonitorFactory,
		CmdOptionDescription:     "bar option",
	}

	Register("foo", fooMonitorHandler)
	Register("bar", barMonitorHandler)

	expectedProblemDaemonNames := []types.ProblemDaemonType{"foo", "bar"}
	problemDaemonNames := GetProblemDaemonNames()

	assert.ElementsMatch(t, expectedProblemDaemonNames, problemDaemonNames)
	assert.Equal(t, "foo option", GetProblemDaemonHandlerOrDie("foo").CmdOptionDescription)
	assert.Equal(t, "bar option", GetProblemDaemonHandlerOrDie("bar").CmdOptionDescription)

	handlers = make(map[types.ProblemDaemonType]types.ProblemDaemonHandler)
}

func TestGetProblemDaemonHandlerOrDie(t *testing.T) {
	fooMonitorFactory := func(configPath string) types.Monitor {
		return nil
	}
	fooMonitorHandler := types.ProblemDaemonHandler{
		CreateProblemDaemonOrDie: fooMonitorFactory,
		CmdOptionDescription:     "foo option",
	}

	Register("foo", fooMonitorHandler)

	assert.NotPanics(t, func() { GetProblemDaemonHandlerOrDie("foo") })
	assert.Panics(t, func() { GetProblemDaemonHandlerOrDie("bar") })

	handlers = make(map[types.ProblemDaemonType]types.ProblemDaemonHandler)
}
