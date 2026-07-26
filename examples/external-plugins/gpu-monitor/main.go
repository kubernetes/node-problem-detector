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

// gpu-monitor is an example external monitor plugin for NPD that monitors GPU health.
// This example demonstrates how to implement the ExternalMonitor gRPC service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "k8s.io/node-problem-detector/api/services/external/v1"
)

var (
	socketPath           = flag.String("socket", "/var/run/npd/gpu-monitor.sock", "Unix socket path for gRPC server")
	temperatureThreshold = flag.Int("temp-threshold", 85, "Temperature threshold in Celsius")
	memoryThreshold      = flag.Float64("memory-threshold", 95.0, "Memory usage threshold in percentage")
	version              = flag.String("version", "1.0.0", "Monitor version")
)

// GPUMonitor implements the ExternalMonitor gRPC service.
type GPUMonitor struct {
	pb.UnimplementedExternalMonitorServer

	tempThreshold int
	memThreshold  float64
	version       string
	shutdownChan  chan struct{}
}

// GPUStats represents GPU statistics.
type GPUStats struct {
	Temperature   int
	MemoryUsed    int
	MemoryTotal   int
	MemoryPercent float64
	PowerUsage    int
	Available     bool
	ErrorMessage  string
}

// NewGPUMonitor creates a new GPU monitor instance.
func NewGPUMonitor(tempThreshold int, memThreshold float64, version string) *GPUMonitor {
	return &GPUMonitor{
		tempThreshold: tempThreshold,
		memThreshold:  memThreshold,
		version:       version,
		shutdownChan:  make(chan struct{}),
	}
}

// CheckHealth implements the ExternalMonitor.CheckHealth gRPC method.
func (m *GPUMonitor) CheckHealth(ctx context.Context, req *pb.HealthCheckRequest) (*pb.Status, error) {
	log.Printf("CheckHealth called (sequence: %d)", req.Sequence)

	// Check for parameter overrides
	tempThreshold := m.tempThreshold
	memThreshold := m.memThreshold

	if threshold, ok := req.Parameters["temperature_threshold"]; ok {
		if val, err := strconv.Atoi(threshold); err == nil {
			tempThreshold = val
		}
	}

	if threshold, ok := req.Parameters["memory_threshold"]; ok {
		if val, err := strconv.ParseFloat(threshold, 64); err == nil {
			memThreshold = val
		}
	}

	// Get GPU statistics
	stats, err := m.getGPUStats(ctx)
	if err != nil {
		log.Printf("Failed to get GPU stats: %v", err)
		// Return status indicating monitoring error
		return &pb.Status{
			Source: "gpu-monitor",
			Conditions: []*pb.Condition{
				{
					Type:       "GPUHealthy",
					Status:     pb.ConditionStatus_CONDITION_STATUS_UNKNOWN,
					Transition: timestamppb.Now(),
					Reason:     "GPUMonitoringError",
					Message:    fmt.Sprintf("Failed to monitor GPU: %v", err),
				},
			},
		}, nil
	}

	// Check if GPU is available
	if !stats.Available {
		return &pb.Status{
			Source: "gpu-monitor",
			Events: []*pb.Event{
				{
					Severity:  pb.Severity_SEVERITY_WARN,
					Timestamp: timestamppb.Now(),
					Reason:    "GPUNotAvailable",
					Message:   "No GPU detected or nvidia-smi not available",
				},
			},
			Conditions: []*pb.Condition{
				{
					Type:       "GPUHealthy",
					Status:     pb.ConditionStatus_CONDITION_STATUS_UNKNOWN,
					Transition: timestamppb.Now(),
					Reason:     "GPUNotAvailable",
					Message:    "GPU not available for monitoring",
				},
			},
		}, nil
	}

	// Analyze GPU health
	events := []*pb.Event{}
	isHealthy := true
	var reason, message string

	// Check temperature
	if stats.Temperature > tempThreshold {
		isHealthy = false
		reason = "GPUOverheating"
		message = fmt.Sprintf("GPU temperature %d°C exceeds threshold %d°C", stats.Temperature, tempThreshold)

		events = append(events, &pb.Event{
			Severity:  pb.Severity_SEVERITY_WARN,
			Timestamp: timestamppb.Now(),
			Reason:    "GPUOverheating",
			Message:   message,
		})
	}

	// Check memory usage
	if stats.MemoryPercent > memThreshold {
		if !isHealthy {
			reason = "GPUMultipleIssues"
			message = fmt.Sprintf("GPU has multiple issues: temperature=%d°C, memory=%.1f%%", stats.Temperature, stats.MemoryPercent)
		} else {
			isHealthy = false
			reason = "GPUMemoryHigh"
			message = fmt.Sprintf("GPU memory usage %.1f%% exceeds threshold %.1f%%", stats.MemoryPercent, memThreshold)
		}

		events = append(events, &pb.Event{
			Severity:  pb.Severity_SEVERITY_WARN,
			Timestamp: timestamppb.Now(),
			Reason:    "GPUMemoryHigh",
			Message:   fmt.Sprintf("GPU memory usage %.1f%% exceeds threshold %.1f%%", stats.MemoryPercent, memThreshold),
		})
	}

	// Set healthy status
	if isHealthy {
		reason = "GPUIsHealthy"
		message = fmt.Sprintf("GPU is healthy: temp=%d°C, memory=%.1f%%, power=%dW",
			stats.Temperature, stats.MemoryPercent, stats.PowerUsage)
	}

	conditionStatus := pb.ConditionStatus_CONDITION_STATUS_FALSE // Healthy
	if !isHealthy {
		conditionStatus = pb.ConditionStatus_CONDITION_STATUS_TRUE // Problem
	}

	return &pb.Status{
		Source: "gpu-monitor",
		Events: events,
		Conditions: []*pb.Condition{
			{
				Type:       "GPUHealthy",
				Status:     conditionStatus,
				Transition: timestamppb.Now(),
				Reason:     reason,
				Message:    message,
			},
		},
	}, nil
}

// GetMetadata implements the ExternalMonitor.GetMetadata gRPC method.
func (m *GPUMonitor) GetMetadata(ctx context.Context, req *emptypb.Empty) (*pb.MonitorMetadata, error) {
	log.Println("GetMetadata called")

	return &pb.MonitorMetadata{
		Name:                "gpu-monitor",
		Version:             m.version,
		Description:         "Monitors NVIDIA GPU health including temperature and memory usage",
		SupportedConditions: []string{"GPUHealthy"},
		Capabilities: map[string]string{
			"temperature_monitoring": "true",
			"memory_monitoring":      "true",
			"power_monitoring":       "true",
			"nvidia_smi_required":    "true",
		},
		ApiVersion: "v1",
	}, nil
}

// Stop implements the ExternalMonitor.Stop gRPC method.
func (m *GPUMonitor) Stop(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	log.Println("Stop called - initiating graceful shutdown")

	close(m.shutdownChan)
	return &emptypb.Empty{}, nil
}

// getGPUStats retrieves GPU statistics using nvidia-smi.
func (m *GPUMonitor) getGPUStats(ctx context.Context) (*GPUStats, error) {
	// Check if nvidia-smi is available
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return &GPUStats{Available: false}, nil
	}

	// Run nvidia-smi to get GPU stats
	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=temperature.gpu,memory.used,memory.total,power.draw",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi execution failed: %v", err)
	}

	// Parse output
	line := strings.TrimSpace(string(output))
	if line == "" {
		return &GPUStats{Available: false}, nil
	}

	// Split by comma and parse values
	parts := strings.Split(line, ",")
	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected nvidia-smi output format: %s", line)
	}

	stats := &GPUStats{Available: true}

	// Parse temperature
	if temp, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
		stats.Temperature = temp
	}

	// Parse memory
	if memUsed, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
		stats.MemoryUsed = memUsed
	}
	if memTotal, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
		stats.MemoryTotal = memTotal
	}

	// Calculate memory percentage
	if stats.MemoryTotal > 0 {
		stats.MemoryPercent = float64(stats.MemoryUsed) / float64(stats.MemoryTotal) * 100.0
	}

	// Parse power (might contain "N/A")
	powerStr := strings.TrimSpace(parts[3])
	if powerStr != "N/A" {
		// Remove any non-digit characters except decimal point
		re := regexp.MustCompile(`[^\d.]`)
		powerStr = re.ReplaceAllString(powerStr, "")
		if power, err := strconv.ParseFloat(powerStr, 64); err == nil {
			stats.PowerUsage = int(power)
		}
	}

	log.Printf("GPU stats: temp=%d°C, memory=%d/%dMB (%.1f%%), power=%dW",
		stats.Temperature, stats.MemoryUsed, stats.MemoryTotal, stats.MemoryPercent, stats.PowerUsage)

	return stats, nil
}

func main() {
	flag.Parse()

	log.Printf("Starting GPU Monitor v%s", *version)
	log.Printf("Socket: %s", *socketPath)
	log.Printf("Temperature threshold: %d°C", *temperatureThreshold)
	log.Printf("Memory threshold: %.1f%%", *memoryThreshold)

	// Create monitor instance
	monitor := NewGPUMonitor(*temperatureThreshold, *memoryThreshold, *version)

	// Remove existing socket file
	if err := os.RemoveAll(*socketPath); err != nil {
		log.Fatalf("Failed to remove existing socket: %v", err)
	}

	// Create Unix socket listener
	lc := &net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "unix", *socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on socket %s: %v", *socketPath, err)
	}
	defer func() { _ = listener.Close() }()

	// Set socket permissions (readable/writable by owner and group)
	if err := os.Chmod(*socketPath, 0o660); err != nil {
		log.Printf("Warning: failed to set socket permissions: %v", err)
	}

	// Create gRPC server
	server := grpc.NewServer()
	pb.RegisterExternalMonitorServer(server, monitor)

	log.Printf("GPU Monitor listening on %s", *socketPath)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	// Wait for shutdown signal or server error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal")
	case <-monitor.shutdownChan:
		log.Println("Received shutdown via gRPC")
	case err := <-serverErr:
		if err != nil {
			log.Printf("Server error: %v", err)
		}
	}

	// Graceful shutdown
	log.Println("Shutting down GPU Monitor...")
	server.GracefulStop()

	// Clean up socket file
	_ = os.RemoveAll(*socketPath)
	log.Println("GPU Monitor stopped")
}
