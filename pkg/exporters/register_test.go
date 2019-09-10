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

package exporters

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/node-problem-detector/pkg/types"
)

func TestRegistration(t *testing.T) {
	fooExporterFactory := func(types.CommandLineOptions) types.Exporter {
		return nil
	}
	fooExporterHandler := types.ExporterHandler{
		CreateExporterOrDie: fooExporterFactory,
		Options:             nil,
	}

	barExporterFactory := func(types.CommandLineOptions) types.Exporter {
		return nil
	}
	barExporterHandler := types.ExporterHandler{
		CreateExporterOrDie: barExporterFactory,
		Options:             nil,
	}

	Register("foo", fooExporterHandler)
	Register("bar", barExporterHandler)

	expectedExporterNames := []types.ExporterType{"foo", "bar"}
	exporterNames := GetExporterNames()
	assert.ElementsMatch(t, expectedExporterNames, exporterNames)

	handlers = make(map[types.ExporterType]types.ExporterHandler)
}

func TestGetExporterHandlerOrDie(t *testing.T) {
	fooExporterFactory := func(types.CommandLineOptions) types.Exporter {
		return nil
	}
	fooExporterHandler := types.ExporterHandler{
		CreateExporterOrDie: fooExporterFactory,
		Options:             nil,
	}

	Register("foo", fooExporterHandler)

	assert.NotPanics(t, func() { GetExporterHandlerOrDie("foo") })
	assert.Panics(t, func() { GetExporterHandlerOrDie("bar") })

	handlers = make(map[types.ExporterType]types.ExporterHandler)
}
