# APIServer Monitor

*APIServer Monitor* is a problem daemon in node-problem-detector that periodically checks connectivity to the Kubernetes API server (`GET /version`).

Unlike regular Kubernetes clients that reuse an existing HTTP/2 connection, each poll establishes a fresh TCP+TLS connection to `kube-apiserver` and closes idle connections afterward so that network path issues (e.g., firewall rules blocking new connections or broken load balancer backends) are detected.

## Scale Considerations

Because each check performs a full TCP and TLS handshake against `kube-apiserver`, choose `invokeInterval` with cluster size in mind. For example, a `1m` interval across a 5,000-node cluster generates ~83 handshakes/s against the control plane.

## Configuration

See [`config/apiserver-monitor.json`](../../config/apiserver-monitor.json) for an example configuration:

```json
{
  "invokeInterval": "1m",
  "timeout": "30s",
  "source": "apiserver-monitor",
  "metricsReporting": true,
  "metricsOnly": true,
  "conditions": [
    {
      "type": "APIServerUnreachable",
      "reason": "APIServerIsReachable",
      "message": "apiserver is reachable"
    }
  ]
}
```

### Configuration Fields

* `invokeInterval`: Interval between connectivity checks (default: `"1m"`). Must be greater than `0s`.
* `timeout`: Per-check request timeout (default: `"30s"`). Must be greater than `0s` and at most `invokeInterval`.
* `source`: Source name reported in statuses and events (default: `"apiserver-monitor"`).
* `metricsReporting`: Whether to report problem metrics (`problem_counter` and `problem_gauge`; default: `true`).
* `metricsOnly`: When `true`, suppresses emitting statuses to exporters so node-problem-detector does not attempt to write node conditions and events to the Kubernetes `Node` object when `--enable-k8s-exporter` is enabled (which is likely to fail while the API server is unreachable anyway); only problem metrics are updated (default: `false` in code, `true` in `config/apiserver-monitor.json`). Requires `metricsReporting` to be `true`.
* `skipInitialStatus`: When `true`, skips sending the initial default status on startup (default: `false`).
* `conditions`: List of node conditions managed by the monitor (defaults to `APIServerUnreachable` with default reason `APIServerIsReachable` and message `"apiserver is reachable"`).

## Metrics

When `metricsReporting` is enabled, the monitor records:

* `problem_counter{reason="APIServerUnreachable"}`: Incremented each time a condition transitions from `False` to `True`.
* `problem_gauge{type="APIServerUnreachable",reason="APIServerUnreachable"}`: Set to `1` while the API server check is failing and `0` when reachable.
