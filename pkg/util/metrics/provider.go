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
package metrics

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	meterProvider *sdkmetric.MeterProvider
	readers       []sdkmetric.Reader
	resources     []*resource.Resource
	readersMutex  sync.Mutex
	initialized   bool
)

// AddReader adds a metric reader to be used when setting up the meter provider.
// This should be called before any metrics are created.
func AddReader(reader sdkmetric.Reader) {
	readersMutex.Lock()
	defer readersMutex.Unlock()
	readers = append(readers, reader)
}

// AddReaderWithResource adds a metric reader with a specific resource to be used when setting up the meter provider.
// This should be called before any metrics are created.
func AddReaderWithResource(reader sdkmetric.Reader, res *resource.Resource) {
	readersMutex.Lock()
	defer readersMutex.Unlock()
	readers = append(readers, reader)
	resources = append(resources, res)
}

// SetupMeterProvider initializes the global meter provider with all registered readers.
// This should be called after all readers have been added.
// Can be called multiple times safely - only the first call takes effect.
func SetupMeterProvider() {
	readersMutex.Lock()
	defer readersMutex.Unlock()

	if initialized {
		return
	}
	initialized = true

	opts := make([]sdkmetric.Option, 0, len(readers)+1)
	for _, reader := range readers {
		opts = append(opts, sdkmetric.WithReader(reader))
	}

	// Merge all resources if any
	if len(resources) > 0 {
		merged := resources[0]
		for i := 1; i < len(resources); i++ {
			var err error
			merged, err = resource.Merge(merged, resources[i])
			if err != nil {
				// If merge fails, continue with what we have
				continue
			}
		}
		opts = append(opts, sdkmetric.WithResource(merged))
	}

	meterProvider = sdkmetric.NewMeterProvider(opts...)
	otel.SetMeterProvider(meterProvider)
}

// GetMeter returns the global meter for creating metrics.
// Note: Metrics created before SetupMeterProvider() is called will use a no-op meter
// and won't be exported. Always call SetupMeterProvider() after adding all readers.
func GetMeter() metric.Meter {
	return otel.Meter("k8s.io/node-problem-detector")
}

// ShutdownMeterProvider gracefully shuts down the meter provider.
func ShutdownMeterProvider() error {
	readersMutex.Lock()
	defer readersMutex.Unlock()
	if meterProvider != nil {
		return meterProvider.Shutdown(context.Background())
	}
	return nil
}
