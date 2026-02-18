//go:build !disable_stackdriver_exporter

// Copyright 2024 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudmock

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// MetadataTestServer is a mock GCE metadata server for testing.
// It simulates the GCE metadata service that returns instance metadata.
type MetadataTestServer struct {
	server         *httptest.Server
	projectID      string
	zone           string
	instanceID     string
	instanceName   string
	requestCount   map[string]int
	failureCount   int
	maxFailures    int
	mu             sync.Mutex
}

// NewMetadataTestServer creates a new mock GCE metadata server with the specified values.
func NewMetadataTestServer(projectID, zone, instanceID, instanceName string) *MetadataTestServer {
	mts := &MetadataTestServer{
		projectID:    projectID,
		zone:         zone,
		instanceID:   instanceID,
		instanceName: instanceName,
		requestCount: make(map[string]int),
	}

	mts.server = httptest.NewServer(http.HandlerFunc(mts.handleRequest))
	return mts
}

// SetTransientFailures configures the server to fail the first N requests to test retry logic.
func (m *MetadataTestServer) SetTransientFailures(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxFailures = count
	m.failureCount = 0
}

// RequestCount returns the number of times a specific endpoint was called.
func (m *MetadataTestServer) RequestCount(endpoint string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestCount[endpoint]
}

func (m *MetadataTestServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.requestCount[r.URL.Path]++

	// Simulate transient failures for retry testing
	if m.failureCount < m.maxFailures {
		m.failureCount++
		m.mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	m.mu.Unlock()

	// Check for required Metadata-Flavor header
	if r.Header.Get("Metadata-Flavor") != "Google" {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Missing Metadata-Flavor: Google header")
		return
	}

	// Route to appropriate handler based on path
	switch {
	case strings.HasSuffix(r.URL.Path, "/project/project-id"):
		fmt.Fprint(w, m.projectID)
	case strings.HasSuffix(r.URL.Path, "/instance/zone"):
		// Zone is returned as projects/PROJECT_NUMBER/zones/ZONE
		fmt.Fprintf(w, "projects/12345/zones/%s", m.zone)
	case strings.HasSuffix(r.URL.Path, "/instance/id"):
		fmt.Fprint(w, m.instanceID)
	case strings.HasSuffix(r.URL.Path, "/instance/name"):
		fmt.Fprint(w, m.instanceName)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Unknown metadata endpoint: %s", r.URL.Path)
	}
}

// Close shuts down the mock metadata server.
func (m *MetadataTestServer) Close() {
	m.server.Close()
}

// Endpoint returns the base URL of the mock metadata server.
func (m *MetadataTestServer) Endpoint() string {
	return m.server.URL
}

// Host returns just the host:port portion suitable for GCE_METADATA_HOST env var.
func (m *MetadataTestServer) Host() string {
	// Strip the http:// prefix
	return strings.TrimPrefix(m.server.URL, "http://")
}
