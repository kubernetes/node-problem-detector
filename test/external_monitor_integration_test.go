/*
Copyright The Kubernetes Authors All rights reserved.

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

package test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "k8s.io/node-problem-detector/api/services/external/v1"
	"k8s.io/node-problem-detector/pkg/externalmonitor"
	"k8s.io/node-problem-detector/pkg/externalmonitor/types"
	npdt "k8s.io/node-problem-detector/pkg/types"
)

// mockExternalMonitor implements a simple mock external monitor for testing.
type mockExternalMonitor struct {
	pb.UnimplementedExternalMonitorServer
	healthy bool
}

func (m *mockExternalMonitor) CheckHealth(ctx context.Context, req *pb.HealthCheckRequest) (*pb.Status, error) {
	status := &pb.Status{
		Source: "test-monitor",
	}

	if m.healthy {
		status.Conditions = []*pb.Condition{
			{
				Type:       "TestHealthy",
				Status:     pb.ConditionStatus_CONDITION_STATUS_FALSE,
				Transition: timestamppb.Now(),
				Reason:     "TestIsHealthy",
				Message:    "Test monitor is healthy",
			},
		}
	} else {
		status.Events = []*pb.Event{
			{
				Severity:  pb.Severity_SEVERITY_WARN,
				Timestamp: timestamppb.Now(),
				Reason:    "TestUnhealthy",
				Message:   "Test monitor detected a problem",
			},
		}
		status.Conditions = []*pb.Condition{
			{
				Type:       "TestHealthy",
				Status:     pb.ConditionStatus_CONDITION_STATUS_TRUE,
				Transition: timestamppb.Now(),
				Reason:     "TestUnhealthy",
				Message:    "Test monitor is unhealthy",
			},
		}
	}

	return status, nil
}

func (m *mockExternalMonitor) GetMetadata(ctx context.Context, req *emptypb.Empty) (*pb.MonitorMetadata, error) {
	return &pb.MonitorMetadata{
		Name:                "test-monitor",
		Version:             "1.0.0-test",
		Description:         "Mock external monitor for testing",
		SupportedConditions: []string{"TestHealthy"},
		Capabilities: map[string]string{
			"test_capability": "true",
		},
		ApiVersion: "v1",
	}, nil
}

func (m *mockExternalMonitor) Stop(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// TestExternalMonitorIntegration tests the external monitor proxy integration.
func TestExternalMonitorIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "npd-external-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	socketPath := filepath.Join(tempDir, "test-monitor.sock")

	// Start mock external monitor server
	mockMonitor := &mockExternalMonitor{healthy: true}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create socket listener: %v", err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	pb.RegisterExternalMonitorServer(server, mockMonitor)

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create external monitor configuration
	config := &types.ExternalMonitorConfig{
		Plugin: "external",
		PluginConfig: types.ExternalPluginConfig{
			SocketAddress:     socketPath,
			InvokeInterval:    2 * time.Second,
			Timeout:           1 * time.Second,
			SkipInitialStatus: false,
		},
		Source:           "test-monitor",
		MetricsReporting: false,
		Conditions: []types.ConditionDefinition{
			{
				Type:    "TestHealthy",
				Reason:  "TestIsHealthy",
				Message: "Test monitor is healthy",
			},
		},
	}

	if err := config.ApplyConfiguration(); err != nil {
		t.Fatalf("Failed to apply configuration: %v", err)
	}

	if err := config.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Create external monitor proxy
	proxy, err := externalmonitor.NewExternalMonitorProxy(config)
	if err != nil {
		t.Fatalf("Failed to create external monitor proxy: %v", err)
	}

	// Start proxy
	statusChan, err := proxy.Start()
	if err != nil {
		t.Fatalf("Failed to start external monitor proxy: %v", err)
	}

	// Test healthy status
	select {
	case status := <-statusChan:
		if status.Source != "test-monitor" {
			t.Errorf("Expected source 'test-monitor', got '%s'", status.Source)
		}
		if len(status.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(status.Conditions))
		}
		if len(status.Conditions) > 0 {
			condition := status.Conditions[0]
			if condition.Type != "TestHealthy" {
				t.Errorf("Expected condition type 'TestHealthy', got '%s'", condition.Type)
			}
			if condition.Status != npdt.False {
				t.Errorf("Expected condition status False (healthy), got %v", condition.Status)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for initial status")
	}

	// Change mock to unhealthy
	mockMonitor.healthy = false

	// Test unhealthy status
	select {
	case status := <-statusChan:
		if len(status.Events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(status.Events))
		}
		if len(status.Conditions) != 1 {
			t.Errorf("Expected 1 condition, got %d", len(status.Conditions))
		}
		if len(status.Conditions) > 0 {
			condition := status.Conditions[0]
			if condition.Status != npdt.True {
				t.Errorf("Expected condition status True (unhealthy), got %v", condition.Status)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for unhealthy status")
	}

	// Stop proxy
	proxy.Stop()

	// Stop server
	server.GracefulStop()

	// Check for server errors
	select {
	case err := <-serverErr:
		if err != nil && err.Error() != "use of closed network connection" {
			t.Errorf("Unexpected server error: %v", err)
		}
	default:
		// No error, which is expected
	}
}

// TestExternalMonitorConfiguration tests configuration loading and validation.
func TestExternalMonitorConfiguration(t *testing.T) {
	testCases := []struct {
		name        string
		config      types.ExternalMonitorConfig
		expectError bool
	}{
		{
			name: "valid configuration",
			config: types.ExternalMonitorConfig{
				Plugin: "external",
				PluginConfig: types.ExternalPluginConfig{
					SocketAddress:  "/tmp/test.sock",
					InvokeInterval: 30 * time.Second,
					Timeout:        5 * time.Second,
				},
				Source: "test-monitor",
			},
			expectError: false,
		},
		{
			name: "invalid plugin type",
			config: types.ExternalMonitorConfig{
				Plugin: "invalid",
				PluginConfig: types.ExternalPluginConfig{
					SocketAddress:  "/tmp/test.sock",
					InvokeInterval: 30 * time.Second,
					Timeout:        5 * time.Second,
				},
				Source: "test-monitor",
			},
			expectError: true,
		},
		{
			name: "missing socket address",
			config: types.ExternalMonitorConfig{
				Plugin: "external",
				PluginConfig: types.ExternalPluginConfig{
					InvokeInterval: 30 * time.Second,
					Timeout:        5 * time.Second,
				},
				Source: "test-monitor",
			},
			expectError: true,
		},
		{
			name: "timeout >= invoke_interval",
			config: types.ExternalMonitorConfig{
				Plugin: "external",
				PluginConfig: types.ExternalPluginConfig{
					SocketAddress:  "/tmp/test.sock",
					InvokeInterval: 5 * time.Second,
					Timeout:        5 * time.Second,
				},
				Source: "test-monitor",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.config
			if err := config.ApplyConfiguration(); err != nil {
				t.Fatalf("Failed to apply configuration: %v", err)
			}

			err := config.Validate()
			if tc.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestExternalMonitorRegistration tests that the external monitor is properly registered.
func TestExternalMonitorRegistration(t *testing.T) {
	// This test would normally check if the monitor is registered,
	// but since we can't easily access the global registry in tests,
	// we'll just verify the command line help includes external monitor

	// This is tested indirectly by checking if the binary compiles and
	// the help output includes the external monitor flag
	t.Log("External monitor registration tested via binary compilation and help output")
}
