# ReadonlyFilesystem Recovery Plugin Monitor

## Overview

The `readonly-recovery-plugin-monitor` is a CustomPluginMonitor that detects read-only filesystem conditions and **automatically clears the condition when the filesystem recovers**.

### Why Use This Instead of the Built-in readonly-monitor.json?

The built-in `readonly-monitor.json` uses SystemLogMonitor which:
- Detects when a filesystem becomes read-only
- **Never clears the condition** even after the volume recovers or is unmounted
- Only killing the NPD POD will clear the condition, as the new POD re-initializes the ReadonlyFilesystem condition on the nodes.

This behavior can impact Kubernetes environments using persistent storage backends. For example, with Portworx, volumes may temporarily transition to read-only due to transient I/O errors but recover automatically. However, the node condition can remain True indefinitely for ReadonlyFilesystem.

The `readonly-recovery-plugin-monitor` solves this by:
- Detecting read-only filesystem events from `/dev/kmsg` similar to the built-in monitor
- Additionally, checking current mount state to verify if devices are still read-only
- **Automatically clearing the condition** when all devices recover

## Prerequisites

### Required Volume Mount

The plugin requires access to the host's `/proc` filesystem to check mount states. Add this volume mount to your DaemonSet:

| Volume | Host Path | Mount Path | Purpose |
|--------|-----------|------------|---------|
| `hostproc` | `/proc` | `/host/proc` | Access host's mount table via `/host/proc/1/mounts` |

> **Note:** The `/dev/kmsg` mount is already included in the default DaemonSet.

## Deployment Instructions

### Step 1: Modify the DaemonSet

The plugin files are included in the NPD image at:
- `/config/readonly-recovery-plugin-monitor.json`
- `/config/plugin/check_ro_filesystem.sh`

**Add volume mount to the container spec:**

```yaml
spec:
  containers:
  - name: node-problem-detector
    volumeMounts:
    # ... existing mounts ...
    - name: hostproc
      mountPath: /host/proc
      readOnly: true
```

**Add volume to the pod spec:**

```yaml
spec:
  volumes:
  # ... existing volumes ...
  - name: hostproc
    hostPath:
      path: /proc
```

**Update the command arguments:**

Replace `readonly-monitor.json` with the new plugin:

```yaml
spec:
  containers:
  - name: node-problem-detector
    command:
    - /node-problem-detector
    - --logtostderr
    - --config.system-log-monitor=/config/kernel-monitor.json,/config/docker-monitor.json
    - --config.custom-plugin-monitor=/config/readonly-recovery-plugin-monitor.json
```

> **Note:** Remove `/config/readonly-monitor.json` from `--config.system-log-monitor` to avoid duplicate ReadonlyFilesystem conditions.

### Step 2: Apply Changes

```bash
kubectl apply -f deployment/node-problem-detector.yaml
```

### Step 3: Restart DaemonSet Pods

```bash
kubectl rollout restart daemonset node-problem-detector -n kube-system
```

## Troubleshooting

### Condition Not Updating

1. Check script exists in the pod:
   ```bash
   kubectl exec -n kube-system $POD -- ls -la /config/plugin/check_ro_filesystem.sh
   ```

2. Verify `/host/proc` is mounted:
   ```bash
   kubectl exec -n kube-system $POD -- cat /host/proc/1/mounts | head -5
   ```

3. Check NPD logs for errors:
   ```bash
   kubectl logs -n kube-system $POD | grep -i error
   ```

### Simulate a Read-Only Event (Testing)

On the node, inject a test message into kmsg:
```bash
echo "EXT4-fs (test-device): Remounting filesystem read-only" > /dev/kmsg
```

Then verify NPD detects it within 30 seconds.

## Configuration Options

The plugin configuration in `readonly-recovery-plugin-monitor.json`:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `invoke_interval` | `30s` | How often to run the check script |
| `timeout` | `25s` | Maximum time for script execution |
| `max_output_length` | `512` | Maximum message length in node condition |

