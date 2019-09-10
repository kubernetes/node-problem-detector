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
	"fmt"

	"k8s.io/node-problem-detector/pkg/types"
)

var (
	handlers = make(map[types.ExporterType]types.ExporterHandler)
)

// Register registers a exporter factory method, which will be used to create the exporter.
func Register(exporterType types.ExporterType, handler types.ExporterHandler) {
	handlers[exporterType] = handler
}

// GetExporterNames retrieves all available exporter types.
func GetExporterNames() []types.ExporterType {
	exporterTypes := []types.ExporterType{}
	for exporterType := range handlers {
		exporterTypes = append(exporterTypes, exporterType)
	}
	return exporterTypes
}

// GetExporterHandlerOrDie retrieves the ExporterHandler for a specific type of exporter, panic if error occurs..
func GetExporterHandlerOrDie(exporterType types.ExporterType) types.ExporterHandler {
	handler, ok := handlers[exporterType]
	if !ok {
		panic(fmt.Sprintf("Exporter handler for %v does not exist", exporterType))
	}
	return handler
}

// NewExporters creates all exporters based on the configurations initialized.
func NewExporters() []types.Exporter {
	exporters := []types.Exporter{}
	for _, handler := range handlers {
		exporter := handler.CreateExporterOrDie(handler.Options)
		if exporter == nil {
			continue
		}
		exporters = append(exporters, exporter)
	}
	return exporters
}
