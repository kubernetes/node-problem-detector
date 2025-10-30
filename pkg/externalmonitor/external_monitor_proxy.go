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

package externalmonitor

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/klog/v2"

	pb "k8s.io/node-problem-detector/api/services/external/v1"
	"k8s.io/node-problem-detector/pkg/externalmonitor/types"
	npdt "k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/util/tomb"
)

// ExternalMonitorProxy implements the Monitor interface and proxies calls to external gRPC services.
type ExternalMonitorProxy struct {
	name       string
	config     *types.ExternalMonitorConfig
	conn       *grpc.ClientConn
	client     pb.ExternalMonitorClient
	statusChan chan *npdt.Status
	tomb       *tomb.Tomb

	// Connection management
	connectionMutex    sync.RWMutex
	connected          bool
	lastConnectAttempt time.Time
	backoffAttempt     int
	errorCount         int

	// Status tracking
	sequenceNumber int64
	lastStatus     *npdt.Status
	metadata       *pb.MonitorMetadata
}

// NewExternalMonitorProxy creates a new external monitor proxy.
func NewExternalMonitorProxy(config *types.ExternalMonitorConfig) (*ExternalMonitorProxy, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	proxy := &ExternalMonitorProxy{
		name:       config.Source,
		config:     config,
		statusChan: make(chan *npdt.Status, 1000), // Buffer size matches custompluginmonitor
		tomb:       tomb.NewTomb(),
	}

	return proxy, nil
}

// Start implements the Monitor interface. Returns a status channel and starts monitoring.
func (p *ExternalMonitorProxy) Start() (<-chan *npdt.Status, error) {
	klog.Infof("Starting external monitor proxy: %s", p.name)

	// Attempt initial connection
	if err := p.connect(); err != nil {
		klog.Warningf("Initial connection failed for %s: %v", p.name, err)
		// Don't fail startup - will retry in background
	}

	// Start monitoring loop
	go p.monitorLoop()

	// Start health check loop
	go p.healthCheckLoop()

	return p.statusChan, nil
}

// Stop implements the Monitor interface. Performs graceful shutdown.
func (p *ExternalMonitorProxy) Stop() {
	klog.Infof("Stopping external monitor proxy: %s", p.name)

	// Send stop signal to external plugin
	if p.isConnected() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := p.client.Stop(ctx, &emptypb.Empty{}); err != nil {
			klog.Warningf("Failed to send stop signal to %s: %v", p.name, err)
		}
	}

	// Stop internal loops
	p.tomb.Stop()

	// Close connection
	p.connectionMutex.Lock()
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
	p.connectionMutex.Unlock()

	// Close status channel
	close(p.statusChan)

	klog.Infof("External monitor proxy stopped: %s", p.name)
}

// connect establishes gRPC connection to the external plugin.
func (p *ExternalMonitorProxy) connect() error {
	p.connectionMutex.Lock()
	defer p.connectionMutex.Unlock()

	if p.conn != nil {
		_ = p.conn.Close()
	}

	// Create gRPC connection with keepalive
	conn, err := grpc.NewClient(
		"unix://"+p.config.PluginConfig.SocketAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to external monitor %s: %v", p.name, err)
	}

	p.conn = conn
	p.client = pb.NewExternalMonitorClient(conn)
	p.connected = true
	p.backoffAttempt = 0
	p.errorCount = 0

	klog.Infof("Connected to external monitor: %s", p.name)

	// Get metadata from plugin
	if err := p.fetchMetadata(); err != nil {
		klog.Warningf("Failed to fetch metadata from %s: %v", p.name, err)
	}

	return nil
}

// isConnected safely checks connection status.
func (p *ExternalMonitorProxy) isConnected() bool {
	p.connectionMutex.RLock()
	defer p.connectionMutex.RUnlock()

	if p.conn == nil {
		return false
	}

	state := p.conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// fetchMetadata retrieves metadata from the external plugin.
func (p *ExternalMonitorProxy) fetchMetadata() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.PluginConfig.Timeout)
	defer cancel()

	metadata, err := p.client.GetMetadata(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}

	p.metadata = metadata
	klog.Infof("External monitor %s metadata: version=%s, api_version=%s",
		p.name, metadata.Version, metadata.ApiVersion)

	return nil
}

// monitorLoop is the main monitoring loop that calls CheckHealth periodically.
func (p *ExternalMonitorProxy) monitorLoop() {
	defer p.tomb.Done()

	ticker := time.NewTicker(p.config.PluginConfig.InvokeInterval)
	defer ticker.Stop()

	// Send initial status if not skipped
	if !p.config.PluginConfig.SkipInitialStatus {
		p.sendInitialStatus()
	}

	for {
		select {
		case <-ticker.C:
			p.checkHealth()
		case <-p.tomb.Stopping():
			klog.Infof("Monitor loop stopping for %s", p.name)
			return
		}
	}
}

// healthCheckLoop monitors the gRPC connection health.
func (p *ExternalMonitorProxy) healthCheckLoop() {
	ticker := time.NewTicker(p.config.PluginConfig.HealthCheck.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !p.isConnected() {
				p.attemptReconnection()
			}
		case <-p.tomb.Stopping():
			klog.Infof("Health check loop stopping for %s", p.name)
			return
		}
	}
}

// checkHealth calls the external monitor's CheckHealth method.
func (p *ExternalMonitorProxy) checkHealth() {
	if !p.isConnected() {
		klog.V(4).Infof("Skipping health check for %s - not connected", p.name)
		return
	}

	p.sequenceNumber++

	ctx, cancel := context.WithTimeout(context.Background(), p.config.PluginConfig.Timeout)
	defer cancel()

	req := &pb.HealthCheckRequest{
		Parameters: p.config.PluginConfig.PluginParameters,
		Sequence:   p.sequenceNumber,
	}

	status, err := p.client.CheckHealth(ctx, req)
	if err != nil {
		p.handleError(err, "CheckHealth")
		return
	}

	// Convert protobuf status to internal status
	internalStatus, err := p.convertStatus(status)
	if err != nil {
		klog.Errorf("Failed to convert status from %s: %v", p.name, err)
		return
	}

	// Send status if changed or first time
	if p.shouldSendStatus(internalStatus) {
		select {
		case p.statusChan <- internalStatus:
			klog.V(4).Infof("Sent status from %s: %d events, %d conditions",
				p.name, len(internalStatus.Events), len(internalStatus.Conditions))
		case <-p.tomb.Stopping():
			return
		default:
			klog.Warningf("Status channel full for %s, dropping status", p.name)
		}
	}

	p.lastStatus = internalStatus
	p.errorCount = 0 // Reset error count on success
}

// convertStatus converts protobuf Status to internal Status.
func (p *ExternalMonitorProxy) convertStatus(pbStatus *pb.Status) (*npdt.Status, error) {
	if pbStatus == nil {
		return nil, fmt.Errorf("status is nil")
	}

	status := &npdt.Status{
		Source: pbStatus.Source,
	}

	// Convert events
	for _, pbEvent := range pbStatus.Events {
		event := npdt.Event{
			Severity:  convertSeverity(pbEvent.Severity),
			Timestamp: pbEvent.Timestamp.AsTime(),
			Reason:    pbEvent.Reason,
			Message:   pbEvent.Message,
		}
		status.Events = append(status.Events, event)
	}

	// Convert conditions
	for _, pbCondition := range pbStatus.Conditions {
		condition := npdt.Condition{
			Type:       pbCondition.Type,
			Status:     convertConditionStatus(pbCondition.Status),
			Transition: pbCondition.Transition.AsTime(),
			Reason:     pbCondition.Reason,
			Message:    pbCondition.Message,
		}
		status.Conditions = append(status.Conditions, condition)
	}

	return status, nil
}

// convertSeverity converts protobuf Severity to internal Severity.
func convertSeverity(pbSeverity pb.Severity) npdt.Severity {
	switch pbSeverity {
	case pb.Severity_SEVERITY_INFO:
		return npdt.Info
	case pb.Severity_SEVERITY_WARN:
		return npdt.Warn
	default:
		return npdt.Info
	}
}

// convertConditionStatus converts protobuf ConditionStatus to internal ConditionStatus.
func convertConditionStatus(pbStatus pb.ConditionStatus) npdt.ConditionStatus {
	switch pbStatus {
	case pb.ConditionStatus_CONDITION_STATUS_TRUE:
		return npdt.True
	case pb.ConditionStatus_CONDITION_STATUS_FALSE:
		return npdt.False
	case pb.ConditionStatus_CONDITION_STATUS_UNKNOWN:
		return npdt.Unknown
	default:
		return npdt.Unknown
	}
}

// shouldSendStatus determines if the status should be sent.
func (p *ExternalMonitorProxy) shouldSendStatus(status *npdt.Status) bool {
	// Always send first status
	if p.lastStatus == nil {
		return true
	}

	// Send if events exist (events are always sent)
	if len(status.Events) > 0 {
		return true
	}

	// Send if conditions changed
	return !p.conditionsEqual(p.lastStatus.Conditions, status.Conditions)
}

// conditionsEqual checks if two condition slices are equal.
func (p *ExternalMonitorProxy) conditionsEqual(a, b []npdt.Condition) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Type != b[i].Type ||
			a[i].Status != b[i].Status ||
			a[i].Reason != b[i].Reason ||
			a[i].Message != b[i].Message {
			return false
		}
	}

	return true
}

// sendInitialStatus sends initial conditions from configuration.
func (p *ExternalMonitorProxy) sendInitialStatus() {
	if len(p.config.Conditions) == 0 {
		return
	}

	status := &npdt.Status{
		Source: p.config.Source,
	}

	// Create conditions from configuration
	now := time.Now()
	for _, condDef := range p.config.Conditions {
		condition := npdt.Condition{
			Type:       condDef.Type,
			Status:     npdt.False, // Assume healthy initially
			Transition: now,
			Reason:     condDef.Reason,
			Message:    condDef.Message,
		}
		status.Conditions = append(status.Conditions, condition)
	}

	select {
	case p.statusChan <- status:
		klog.V(4).Infof("Sent initial status from %s", p.name)
	case <-p.tomb.Stopping():
		return
	default:
		klog.Warningf("Status channel full for %s, dropping initial status", p.name)
	}

	p.lastStatus = status
}

// handleError handles gRPC errors and implements error counting.
func (p *ExternalMonitorProxy) handleError(err error, operation string) {
	p.errorCount++

	st := status.Convert(err)

	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded:
		klog.V(4).Infof("Transient error in %s.%s: %v", p.name, operation, err)

		// Mark as disconnected for reconnection
		p.connectionMutex.Lock()
		p.connected = false
		p.connectionMutex.Unlock()

	case codes.Unimplemented:
		klog.Infof("Operation %s not implemented by %s", operation, p.name)

	default:
		klog.Warningf("Error in %s.%s: %v", p.name, operation, err)
	}

	// If too many consecutive errors, trigger reconnection
	if p.errorCount >= p.config.PluginConfig.HealthCheck.ErrorThreshold {
		klog.Warningf("Too many errors for %s (%d), triggering reconnection",
			p.name, p.errorCount)
		p.attemptReconnection()
	}
}

// attemptReconnection attempts to reconnect with exponential backoff.
func (p *ExternalMonitorProxy) attemptReconnection() {
	p.connectionMutex.Lock()
	defer p.connectionMutex.Unlock()

	// Don't attempt too frequently
	if time.Since(p.lastConnectAttempt) < time.Second {
		return
	}

	p.lastConnectAttempt = time.Now()

	// Check if we've exceeded max attempts
	if p.backoffAttempt >= p.config.PluginConfig.RetryPolicy.MaxAttempts {
		klog.Errorf("Giving up reconnection for %s after %d attempts",
			p.name, p.backoffAttempt)
		return
	}

	// Calculate backoff delay
	backoff := time.Duration(float64(p.config.PluginConfig.RetryPolicy.InitialBackoff) *
		math.Pow(p.config.PluginConfig.RetryPolicy.BackoffMultiplier, float64(p.backoffAttempt)))

	if backoff > p.config.PluginConfig.RetryPolicy.MaxBackoff {
		backoff = p.config.PluginConfig.RetryPolicy.MaxBackoff
	}

	p.backoffAttempt++

	klog.Infof("Attempting reconnection for %s (attempt %d) in %v",
		p.name, p.backoffAttempt, backoff)

	// Wait for backoff period
	time.Sleep(backoff)

	// Check if socket exists
	if _, err := os.Stat(p.config.PluginConfig.SocketAddress); err != nil {
		klog.V(4).Infof("Socket %s not available for %s: %v",
			p.config.PluginConfig.SocketAddress, p.name, err)
		return
	}

	// Attempt connection
	if err := p.connectUnsafe(); err != nil {
		klog.Warningf("Reconnection failed for %s: %v", p.name, err)
		return
	}

	klog.Infof("Successfully reconnected to %s", p.name)
}

// connectUnsafe is the internal connection method without locking.
func (p *ExternalMonitorProxy) connectUnsafe() error {
	if p.conn != nil {
		_ = p.conn.Close()
	}

	conn, err := grpc.NewClient(
		"unix://"+p.config.PluginConfig.SocketAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return err
	}

	p.conn = conn
	p.client = pb.NewExternalMonitorClient(conn)
	p.connected = true
	p.backoffAttempt = 0
	p.errorCount = 0

	// Fetch metadata
	if err := p.fetchMetadata(); err != nil {
		klog.Warningf("Failed to fetch metadata from %s after reconnection: %v", p.name, err)
	}

	return nil
}
