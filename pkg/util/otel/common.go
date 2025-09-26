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

package otel

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"k8s.io/klog/v2"
)

var (
	globalMeterProvider *sdkmetric.MeterProvider
	meterProviderOnce   sync.Once
	readers             []sdkmetric.Reader
	readersMutex        sync.Mutex
)

// AddMetricReader adds a metric reader to the global meter provider configuration
// This should be called before InitializeMeterProvider()
// Accepts both pull readers (like Prometheus) and push readers (like Stackdriver)
func AddMetricReader(reader sdkmetric.Reader) {
	readersMutex.Lock()
	defer readersMutex.Unlock()
	readers = append(readers, reader)
}

// InitializeMeterProvider creates and sets the global meter provider with all registered readers
// This should be called once after all exporters have registered their readers
func InitializeMeterProvider() *sdkmetric.MeterProvider {
	meterProviderOnce.Do(func() {
		readersMutex.Lock()
		defer readersMutex.Unlock()

		if len(readers) == 0 {
			klog.Warning("No metric readers registered, creating meter provider without readers")
		}

		// Use global resource with proper service identification
		resource := GetResource()

		// Create meter provider options
		opts := []sdkmetric.Option{sdkmetric.WithResource(resource)}

		// Add all registered readers
		for _, reader := range readers {
			opts = append(opts, sdkmetric.WithReader(reader))
		}

		// Create meter provider with all readers and resource
		globalMeterProvider = sdkmetric.NewMeterProvider(opts...)

		// Set as global meter provider
		otel.SetMeterProvider(globalMeterProvider)

		klog.Infof("OpenTelemetry meter provider initialized with %d readers and resource: %v",
			len(readers), resource.Attributes())
	})
	return globalMeterProvider
}

// GetMeterProvider returns the global meter provider, initializing it if necessary
func GetMeterProvider() *sdkmetric.MeterProvider {
	if globalMeterProvider == nil {
		return InitializeMeterProvider()
	}
	return globalMeterProvider
}

// MeterName is the standard meter name used across the application
const MeterName = "node-problem-detector"

// GetGlobalMeter returns the global meter with the standard node-problem-detector name
func GetGlobalMeter() metric.Meter {
	return otel.Meter(MeterName)
}
