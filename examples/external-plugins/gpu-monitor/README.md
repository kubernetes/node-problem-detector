# GPU Monitor - External NPD Plugin Example

This is an example external monitor plugin for Node Problem Detector (NPD) that monitors NVIDIA GPU health.

## Features

- **Temperature Monitoring**: Alerts when GPU temperature exceeds threshold
- **Memory Usage Monitoring**: Alerts when GPU memory usage is too high
- **Power Usage Tracking**: Reports GPU power consumption
- **Configurable Thresholds**: Runtime parameter override support
- **Graceful Shutdown**: Proper cleanup on termination signals

## Requirements

- NVIDIA GPU with driver installed
- `nvidia-smi` command available
- Unix socket access between NPD and GPU monitor

## Building

### Local Build

```bash
# From repository root
go build -o gpu-monitor ./examples/external-plugins/gpu-monitor/
```

### Docker Build

```bash
# Build GPU monitor image
docker build -t npd-gpu-monitor:latest -f examples/external-plugins/gpu-monitor/Dockerfile .
```

## Configuration

The GPU monitor is configured via NPD's external monitor configuration:

```json
{
  "plugin": "external",
  "pluginConfig": {
    "socketAddress": "/var/run/npd/gpu-monitor.sock",
    "invoke_interval": "30s",
    "timeout": "10s",
    "pluginParameters": {
      "temperature_threshold": "85",
      "memory_threshold": "95.0"
    }
  },
  "source": "gpu-monitor",
  "conditions": [
    {
      "type": "GPUHealthy",
      "reason": "GPUIsHealthy",
      "message": "GPU is functioning properly"
    }
  ]
}
```

### Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `socketAddress` | `/var/run/npd/gpu-monitor.sock` | Unix socket path for gRPC communication |
| `invoke_interval` | `30s` | How often NPD calls CheckHealth |
| `timeout` | `10s` | Timeout for gRPC calls |
| `temperature_threshold` | `85` | Temperature threshold in Celsius |
| `memory_threshold` | `95.0` | Memory usage threshold in percentage |

## Running

### Standalone

```bash
# Run GPU monitor directly
./gpu-monitor --socket=/var/run/npd/gpu-monitor.sock --temp-threshold=85 --memory-threshold=95.0
```

### With Docker

```bash
# Create socket directory
mkdir -p /var/run/npd

# Run GPU monitor container
docker run --rm \
  --gpus all \
  -v /var/run/npd:/var/run/npd \
  npd-gpu-monitor:latest \
  --socket=/var/run/npd/gpu-monitor.sock \
  --temp-threshold=80 \
  --memory-threshold=90.0
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: npd-gpu-monitor
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: npd-gpu-monitor
  template:
    metadata:
      labels:
        app: npd-gpu-monitor
    spec:
      hostPID: true
      containers:
      - name: gpu-monitor
        image: npd-gpu-monitor:latest
        args:
        - --socket=/var/run/npd/gpu-monitor.sock
        - --temp-threshold=85
        - --memory-threshold=95.0
        volumeMounts:
        - name: npd-socket
          mountPath: /var/run/npd
        resources:
          limits:
            nvidia.com/gpu: 1
          requests:
            nvidia.com/gpu: 1
      - name: node-problem-detector
        image: registry.k8s.io/node-problem-detector/node-problem-detector:v0.8.19
        args:
        - --config.external-monitor=/config/gpu-monitor.json
        - --logtostderr
        volumeMounts:
        - name: npd-socket
          mountPath: /var/run/npd
        - name: gpu-monitor-config
          mountPath: /config
      volumes:
      - name: npd-socket
        emptyDir: {}
      - name: gpu-monitor-config
        configMap:
          name: gpu-monitor-config
      nodeSelector:
        accelerator: nvidia-tesla-k80  # Adjust for your GPU type
      tolerations:
      - effect: NoSchedule
        operator: Exists
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: gpu-monitor-config
  namespace: kube-system
data:
  gpu-monitor.json: |
    {
      "plugin": "external",
      "pluginConfig": {
        "socketAddress": "/var/run/npd/gpu-monitor.sock",
        "invoke_interval": "30s",
        "timeout": "10s",
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

## Monitoring Output

### Node Conditions

The GPU monitor reports the `GPUHealthy` condition:

```bash
# Check node conditions
kubectl describe node <node-name>

# Look for GPUHealthy condition:
# Type: GPUHealthy
# Status: False (healthy) | True (problem) | Unknown (error)
# Reason: GPUIsHealthy | GPUOverheating | GPUMemoryHigh | GPUMultipleIssues
```

### Events

GPU problems generate Kubernetes events:

```bash
# Check events
kubectl get events --field-selector source=gpu-monitor

# Example events:
# Warning GPUOverheating  GPU temperature 87°C exceeds threshold 85°C
# Warning GPUMemoryHigh   GPU memory usage 96.5% exceeds threshold 95.0%
```

## Troubleshooting

### GPU Monitor Not Starting

1. Check if nvidia-smi is available:
   ```bash
   nvidia-smi
   ```

2. Verify socket directory permissions:
   ```bash
   ls -la /var/run/npd/
   ```

3. Check GPU monitor logs:
   ```bash
   kubectl logs -n kube-system -l app=npd-gpu-monitor
   ```

### NPD Not Connecting

1. Verify external monitor plugin is enabled:
   ```bash
   kubectl logs -n kube-system -l app=node-problem-detector | grep "external-monitor"
   ```

2. Check socket file exists:
   ```bash
   ls -la /var/run/npd/gpu-monitor.sock
   ```

3. Test gRPC connection manually:
   ```bash
   # Install grpcurl
   grpcurl -unix /var/run/npd/gpu-monitor.sock list
   ```

### High Resource Usage

GPU monitoring can consume resources. Adjust monitoring frequency:

```json
{
  "pluginConfig": {
    "invoke_interval": "60s",  // Reduce frequency
    "timeout": "5s"            // Reduce timeout
  }
}
```

## Extending the Example

This example can be extended to monitor:

- Multiple GPUs
- Additional metrics (clock speeds, utilization)
- GPU processes and memory allocation
- CUDA version compatibility
- Driver version monitoring
- Custom alert thresholds per GPU model

## API Reference

The GPU monitor implements the `ExternalMonitor` gRPC service:

- `CheckHealth()`: Returns current GPU health status
- `GetMetadata()`: Returns monitor capabilities and version
- `Stop()`: Initiates graceful shutdown

See the [protobuf definition](../../../api/services/external/v1/external_monitor.proto) for complete API details.