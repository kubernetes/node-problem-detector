# External Plugin Architecture for Node Problem Detector

This document describes the external plugin architecture for Node Problem Detector (NPD), inspired by containerd's external snapshotter architecture.

## Overview

External plugins allow NPD to support monitors and exporters that run as separate processes, communicating via gRPC over Unix domain sockets. This provides several benefits:

- **Language Independence**: Write plugins in any language that supports gRPC
- **Independent Lifecycle**: Plugins can restart without NPD restart
- **Experimentation**: Test new plugins without recompiling NPD
- **Version Isolation**: Use officially released NPD with custom plugins
- **Resource Isolation**: Plugin failures don't crash NPD

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Node Problem Detector                    │
├─────────────────────────────────────────────────────────────┤
│                    Problem Detector                         │
│  ┌─────────────────┐  ┌──────────────────────────────────┐  │
│  │   In-Process    │  │       External Plugins           │  │
│  │    Monitors     │  │                                  │  │
│  │                 │  │  ┌─────────────────────────────┐  │  │
│  │ • SystemLog     │  │  │    ExternalMonitorProxy     │  │  │
│  │ • CustomPlugin  │  │  │                             │  │  │
│  │ • SystemStats   │  │  │   gRPC Client ←─────────────┼──┼──┼─ Unix Socket
│  │                 │  │  │                             │  │  │
│  └─────────────────┘  │  └─────────────────────────────┘  │  │
│                       │                                  │  │
│  ┌─────────────────┐  │  ┌─────────────────────────────┐  │  │
│  │    Exporters    │  │  │   ExternalExporterProxy     │  │  │
│  │                 │  │  │                             │  │  │
│  │ • K8s           │  │  │   gRPC Client ←─────────────┼──┼──┼─ Unix Socket
│  │ • Prometheus    │  │  │                             │  │  │
│  │ • Stackdriver   │  │  └─────────────────────────────┘  │  │
│  └─────────────────┘  └──────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                                                               │
┌─────────────────────────────────────────────────────────────┼───┐
│                External Plugin Processes                    │   │
│                                                             │   │
│  ┌─────────────────────────────────────────────────────────┼─┐ │
│  │              GPU Monitor                                │ │ │
│  │                                                         │ │ │
│  │  ┌───────────────────────────────────────────────────┐  │ │ │
│  │  │              gRPC Server                          │  │ │ │
│  │  │                                                   │  │ │ │
│  │  │  ExternalMonitor Service Implementation          │  │ │ │
│  │  │  • CheckHealth()                                 │  │ │ │
│  │  │  • GetMetadata()                                 │  │ │ │
│  │  │  • Stop()                                        │  │ │ │
│  │  └───────────────────────────────────────────────────┘  │ │ │
│  └─────────────────────────────────────────────────────────┼─┘ │
│                                                             │   │
│  ┌─────────────────────────────────────────────────────────┼─┐ │
│  │             Custom Exporter                             │ │ │
│  │                                                         │ │ │
│  │  ┌───────────────────────────────────────────────────┐  │ │ │
│  │  │              gRPC Server                          │  │ │ │
│  │  │                                                   │  │ │ │
│  │  │  ExternalExporter Service Implementation         │  │ │ │
│  │  │  • ExportProblems()                              │  │ │ │
│  │  │  • GetMetadata()                                 │  │ │ │
│  │  │  • Stop()                                        │  │ │ │
│  │  └───────────────────────────────────────────────────┘  │ │ │
│  └─────────────────────────────────────────────────────────┼─┘ │
└─────────────────────────────────────────────────────────────┘   │
                                                                  │
          ┌───────────────────────────────────────────────────────┘
          │
          │ /var/run/npd/
          ├── gpu-monitor.sock
          └── custom-exporter.sock
```

## gRPC API

External plugins communicate using gRPC services defined in protobuf:

### ExternalMonitor Service

```protobuf
service ExternalMonitor {
    rpc CheckHealth(HealthCheckRequest) returns (Status);
    rpc GetMetadata(google.protobuf.Empty) returns (MonitorMetadata);
    rpc Stop(google.protobuf.Empty) returns (google.protobuf.Empty);
}
```

### ExternalExporter Service

```protobuf
service ExternalExporter {
    rpc ExportProblems(ExportRequest) returns (ExportResponse);
    rpc GetMetadata(google.protobuf.Empty) returns (ExporterMetadata);
    rpc Stop(google.protobuf.Empty) returns (google.protobuf.Empty);
}
```

## Configuration

External monitors are configured via JSON files:

```json
{
  "plugin": "external",
  "pluginConfig": {
    "socketAddress": "/var/run/npd/gpu-monitor.sock",
    "invoke_interval": "30s",
    "timeout": "10s",
    "retryPolicy": {
      "maxAttempts": 5,
      "backoffMultiplier": 2.0,
      "maxBackoff": "5m",
      "initialBackoff": "1s"
    },
    "healthCheck": {
      "interval": "30s",
      "timeout": "5s",
      "errorThreshold": 3
    },
    "pluginParameters": {
      "temperature_threshold": "85",
      "memory_threshold": "95.0"
    }
  },
  "source": "gpu-monitor",
  "metricsReporting": true,
  "conditions": [
    {
      "type": "GPUHealthy",
      "reason": "GPUIsHealthy",
      "message": "GPU is functioning properly"
    }
  ]
}
```

## Key Features

### Connection Management

- **Health Checking**: Monitors gRPC connection health
- **Automatic Reconnection**: Exponential backoff reconnection on failures
- **Error Tracking**: Configurable error thresholds for plugin disabling
- **Socket Monitoring**: Checks for socket file availability

### Lifecycle Management

- **Independent Processes**: Plugins run as separate processes
- **Graceful Shutdown**: Proper cleanup on stop signals
- **Hot Reload**: Plugin restart without NPD restart
- **Resource Cleanup**: Automatic socket file cleanup

### Error Handling

- **Robust Error Handling**: Comprehensive gRPC error code handling
- **Circuit Breaking**: Disable failing plugins automatically
- **Logging**: Detailed error logging and status tracking
- **Fallback Behavior**: Graceful degradation on plugin failures

### Status Integration

- **Native Status Format**: External plugins produce standard NPD Status objects
- **Event Generation**: Support for both temporary events and permanent conditions
- **Condition Management**: Proper condition state tracking and transitions
- **Metrics Integration**: Optional metrics reporting

## Files and Directories

```
k8s.io/node-problem-detector/
├── api/services/external/v1/           # gRPC protobuf definitions
│   ├── external_monitor.proto
│   └── external_exporter.proto
├── pkg/externalmonitor/                # External monitor proxy implementation
│   ├── external_monitor.go             # Plugin registration
│   ├── external_monitor_proxy.go       # gRPC proxy implementation
│   └── types/
│       └── types.go                    # Configuration types
├── cmd/nodeproblemdetector/problemdaemonplugins/
│   └── external_monitor_plugin.go      # Plugin loader
├── examples/external-plugins/          # Example implementations
│   └── gpu-monitor/                    # GPU monitor example
│       ├── main.go
│       ├── config.json
│       ├── Dockerfile
│       └── README.md
├── test/
│   └── external_monitor_integration_test.go  # Integration tests
└── docs/
    └── external-plugins.md             # This documentation
```

## Usage

### 1. Enable External Monitor Plugin

Build NPD with external monitor support (enabled by default):

```bash
go build ./cmd/nodeproblemdetector/
```

### 2. Create External Monitor Configuration

```bash
cat > /etc/npd/external-monitors/gpu-monitor.json << EOF
{
  "plugin": "external",
  "pluginConfig": {
    "socketAddress": "/var/run/npd/gpu-monitor.sock",
    "invoke_interval": "30s",
    "timeout": "10s"
  },
  "source": "gpu-monitor",
  "conditions": [
    {
      "type": "GPUHealthy",
      "reason": "GPUIsHealthy",
      "message": "GPU is healthy"
    }
  ]
}
EOF
```

### 3. Start External Monitor Plugin

```bash
# Build GPU monitor example
go build ./examples/external-plugins/gpu-monitor/

# Start GPU monitor
./gpu-monitor --socket=/var/run/npd/gpu-monitor.sock
```

### 4. Start NPD with External Monitor

```bash
./nodeproblemdetector \
  --config.external-monitor=/etc/npd/external-monitors/gpu-monitor.json \
  --logtostderr \
  --v=2
```

### 5. Verify Operation

```bash
# Check node conditions
kubectl describe node $(hostname) | grep -A 5 "Conditions:"

# Check for GPU-related events
kubectl get events --field-selector source=gpu-monitor
```

## Example Implementation

See the [GPU Monitor example](../examples/external-plugins/gpu-monitor/) for a complete implementation that demonstrates:

- gRPC server setup
- Health check implementation
- Configuration parameter handling
- Error handling and logging
- Docker containerization
- Kubernetes deployment

## Development Guide

### Creating an External Monitor

1. **Implement gRPC Service**: Implement the `ExternalMonitor` service
2. **Handle Configuration**: Support configuration parameters and defaults
3. **Error Handling**: Implement proper error handling and logging
4. **Testing**: Create unit and integration tests
5. **Documentation**: Document configuration options and behavior

### Best Practices

- **Idempotent Operations**: Ensure CheckHealth is safe to call repeatedly
- **Resource Cleanup**: Properly clean up resources on shutdown
- **Error Reporting**: Use appropriate gRPC status codes
- **Socket Permissions**: Set proper Unix socket permissions
- **Logging**: Provide detailed logs for debugging
- **Graceful Degradation**: Handle partial failures gracefully

## Comparison with Existing Plugins

| Feature | In-Process Plugins | External Plugins |
|---------|-------------------|------------------|
| **Language** | Go only | Any language with gRPC |
| **Lifecycle** | Coupled with NPD | Independent |
| **Updates** | Requires NPD restart | Plugin restart only |
| **Resource Isolation** | Shared with NPD | Isolated process |
| **Communication** | Direct function calls | gRPC over Unix socket |
| **Error Impact** | Can crash NPD | Isolated failures |
| **Development** | Requires NPD rebuild | Standalone development |
| **Performance** | Slightly faster | Minimal gRPC overhead |

## Security Considerations

- **Socket Permissions**: Restrict Unix socket access (660 permissions)
- **Input Validation**: Validate all external plugin responses
- **Resource Limits**: Enforce timeout and response size limits
- **Process Isolation**: Run plugins with minimal privileges
- **Error Boundaries**: Isolate plugin failures from NPD core

## Future Enhancements

- **Dynamic Plugin Registration**: API for runtime plugin registration
- **Plugin Marketplace**: Registry for external plugins
- **Plugin Versioning**: Compatibility checking and version management
- **Performance Metrics**: Monitor plugin performance and resource usage
- **Hot Configuration Reload**: Update plugin configuration without restart

## Troubleshooting

### Plugin Not Starting

1. Check socket file permissions
2. Verify gRPC server implementation
3. Check for port conflicts
4. Review plugin logs

### NPD Not Connecting

1. Verify socket address in configuration
2. Check if external monitor plugin is enabled in NPD build
3. Review NPD logs for connection errors
4. Test gRPC connection manually

### Performance Issues

1. Adjust invoke_interval for less frequent checks
2. Reduce timeout values
3. Monitor plugin resource usage
4. Check for network latency issues

This external plugin architecture provides a robust, production-ready foundation for extending NPD with custom monitoring capabilities while maintaining the reliability and performance of the core system.