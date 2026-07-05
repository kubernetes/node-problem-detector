//go:build !disable_stackdriver_exporter

// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudmock

import (
	"context"
	"net"
	"sync"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MetricsTestServer is a mock Google Cloud Monitoring server for testing.
// It captures all metric-related requests and provides methods to retrieve them.
type MetricsTestServer struct {
	lis                  net.Listener
	srv                  *grpc.Server
	endpoint             string
	createTimeSeriesReqs []*monitoringpb.CreateTimeSeriesRequest
	mu                   sync.Mutex
}

// Shutdown gracefully stops the mock server.
func (m *MetricsTestServer) Shutdown() {
	m.srv.GracefulStop()
}

// Endpoint returns the address of the mock server.
func (m *MetricsTestServer) Endpoint() string {
	return m.endpoint
}

// CreateTimeSeriesRequests returns and clears all captured CreateTimeSeries requests.
func (m *MetricsTestServer) CreateTimeSeriesRequests() []*monitoringpb.CreateTimeSeriesRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	reqs := m.createTimeSeriesReqs
	m.createTimeSeriesReqs = nil
	return reqs
}

func (m *MetricsTestServer) appendCreateTimeSeriesReq(_ context.Context, req *monitoringpb.CreateTimeSeriesRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createTimeSeriesReqs = append(m.createTimeSeriesReqs, req)
}

// Serve starts the mock server (blocks until shutdown).
func (m *MetricsTestServer) Serve() error {
	return m.srv.Serve(m.lis)
}

type fakeMetricServiceServer struct {
	monitoringpb.UnimplementedMetricServiceServer
	metricsTestServer *MetricsTestServer
}

// CreateTimeSeries simulates a call to Google Cloud Monitoring.
func (f *fakeMetricServiceServer) CreateTimeSeries(
	ctx context.Context,
	req *monitoringpb.CreateTimeSeriesRequest,
) (*emptypb.Empty, error) {
	f.metricsTestServer.appendCreateTimeSeriesReq(ctx, req)

	pointCount := int32(len(req.TimeSeries))
	statusResp, _ := status.New(codes.OK, "").WithDetails(
		&monitoringpb.CreateTimeSeriesSummary{
			TotalPointCount:   pointCount,
			SuccessPointCount: pointCount,
		})

	return &emptypb.Empty{}, statusResp.Err()
}

// NewMetricTestServer creates and starts a new mock Google Cloud Monitoring server.
// The server listens on a random port on localhost.
func NewMetricTestServer() *MetricsTestServer {
	srv := grpc.NewServer()
	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	testServer := &MetricsTestServer{
		endpoint: lis.Addr().String(),
		lis:      lis,
		srv:      srv,
	}

	monitoringpb.RegisterMetricServiceServer(
		srv,
		&fakeMetricServiceServer{metricsTestServer: testServer},
	)

	go func() {
		_ = testServer.Serve()
	}()

	return testServer
}
