# node-problem-detector

[![Build Status](https://travis-ci.org/kubernetes/node-problem-detector.svg?branch=master)](https://travis-ci.org/kubernetes/node-problem-detector)  [![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes/node-problem-detector)](https://goreportcard.com/report/github.com/kubernetes/node-problem-detector)

node-problem-detector aims to make various node problems visible to the upstream
layers in the cluster management stack.
It is a daemon that runs on each node, detects node
problems and reports them to apiserver.
node-problem-detector can either run as a
[DaemonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) or run standalone.
Now it is running as a
[Kubernetes Addon](https://github.com/kubernetes/kubernetes/tree/master/cluster/addons)
enabled by default in the GKE cluster. It is also enabled by default in AKS as part of the
[AKS Linux Extension](https://learn.microsoft.com/en-us/azure/aks/faq#what-is-the-purpose-of-the-aks-linux-extension-i-see-installed-on-my-linux-vmss-instances).

# Background

There are tons of node problems that could possibly affect the pods running on the
node, such as:

* Infrastructure daemon issues: ntp service down;
* Hardware issues: Bad CPU, memory or disk;
* Kernel issues: Kernel deadlock, corrupted file system;
* Container runtime issues: Unresponsive runtime daemon;
* ...

Currently, these problems are invisible to the upstream layers in the cluster management
stack, so Kubernetes will continue scheduling pods to the bad nodes.

To solve this problem, we introduced this new daemon **node-problem-detector** to
collect node problems from various daemons and make them visible to the upstream
layers. Once upstream layers have visibility to those problems, we can discuss the
[remedy system](#remedy-systems).

# Problem API

node-problem-detector uses `Event` and `NodeCondition` to report problems to
apiserver.

* `NodeCondition`: Permanent problem that makes the node unavailable for pods should
be reported as `NodeCondition`.
* `Event`: Temporary problem that has limited impact on pod but is informative
should be reported as `Event`.

# Problem Daemon

A problem daemon is a sub-daemon of node-problem-detector. It monitors specific
kinds of node problems and reports them to node-problem-detector.

A problem daemon could be:

* A tiny daemon designed for dedicated Kubernetes use-cases.
* An existing node health monitoring daemon integrated with node-problem-detector.

Currently, a problem daemon is running as a goroutine in the node-problem-detector
binary. In the future, we'll separate node-problem-detector and problem daemons into
different containers, and compose them with pod specification.

Each category of problem daemon can be disabled at compilation time by setting
corresponding build tags. If they are disabled at compilation time, then all their
build dependencies, global variables and background goroutines will be trimmed out
of the compiled executable.

List of supported problem daemons types:

| Problem Daemon Types |  NodeCondition  | Description | Configs | Disabling Build Tag |
|----------------|:---------------:|:------------|:--------|:--------------------|
| [SystemLogMonitor](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/systemlogmonitor) | KernelDeadlock ReadonlyFilesystem FrequentKubeletRestart FrequentContainerdRestart | A system log monitor monitors system log and reports problems and metrics according to predefined rules. | [filelog](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor-filelog.json), [kmsg](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json), [kernel](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor-counter.json) [abrt](https://github.com/kubernetes/node-problem-detector/blob/master/config/abrt-adaptor.json) [systemd](https://github.com/kubernetes/node-problem-detector/blob/master/config/systemd-monitor-counter.json) | disable_system_log_monitor
| [SystemStatsMonitor](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/systemstatsmonitor) | None(Could be added in the future) | A system stats monitor for node-problem-detector to collect various health-related system stats as metrics. See the proposal [here](https://docs.google.com/document/d/1SeaUz6kBavI283Dq8GBpoEUDrHA2a795xtw0OvjM568/edit). | [system-stats-monitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json) | disable_system_stats_monitor
| [CustomPluginMonitor](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/custompluginmonitor) | On-demand(According to users configuration), existing example: NTPProblem | A custom plugin monitor for node-problem-detector to invoke and check various node problems with user-defined check scripts. See the proposal [here](https://docs.google.com/document/d/1jK_5YloSYtboj-DtfjmYKxfNnUxCAvohLnsH5aGCAYQ/edit#). | [example](https://github.com/kubernetes/node-problem-detector/blob/4ad49bbd84b8ced45ac825eac01ec93d9235935e/config/custom-plugin-monitor.json) | disable_custom_plugin_monitor
| [HealthChecker](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/healthchecker) | KubeletUnhealthy ContainerRuntimeUnhealthy| A health checker for node-problem-detector to check kubelet and container runtime health. | [kubelet](https://github.com/kubernetes/node-problem-detector/blob/master/config/health-checker-kubelet.json) [containerd](https://github.com/kubernetes/node-problem-detector/blob/master/config/health-checker-containerd.json) |

# Exporter

An exporter is a component of node-problem-detector. It reports node problems and/or metrics to
certain backends. Some of them can be disabled at compile-time using a build tag. List of supported exporters:

| Exporter |Description | Disabling Build Tag |
|----------|:-----------|:--------------------|
| Kubernetes exporter | Kubernetes exporter reports node problems to Kubernetes API server: temporary problems get reported as Events, and permanent problems get reported as Node Conditions. |
| Prometheus exporter | Prometheus exporter reports node problems and metrics locally as Prometheus metrics |
| [Stackdriver exporter](https://github.com/kubernetes/node-problem-detector/blob/master/config/exporter/stackdriver-exporter.json) | Stackdriver exporter reports node problems and metrics to Stackdriver Monitoring API. | disable_stackdriver_exporter

# Usage

## Flags

* `--version`: Print current version of node-problem-detector.
* `--hostname-override`: A customized node name used for node-problem-detector to update conditions and emit events. node-problem-detector gets node name first from `hostname-override`, then `NODE_NAME` environment variable and finally fall back to `os.Hostname`.

#### For System Log Monitor

* `--config.system-log-monitor`: List of paths to system log monitor configuration files, comma-separated, e.g.
  [config/kernel-monitor.json](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json).
  Node problem detector will start a separate log monitor for each configuration. You can
  use different log monitors to monitor different system logs.

#### For System Stats Monitor

* `--config.system-stats-monitor`: List of paths to system stats monitor config files, comma-separated, e.g.
  [config/system-stats-monitor.json](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json).
  Node problem detector will start a separate system stats monitor for each configuration. You can
  use different system stats monitors to monitor different problem-related system stats.

#### For Custom Plugin Monitor

* `--config.custom-plugin-monitor`: List of paths to custom plugin monitor config files, comma-separated, e.g.
  [config/custom-plugin-monitor.json](https://github.com/kubernetes/node-problem-detector/blob/master/config/custom-plugin-monitor.json).
  Node problem detector will start a separate custom plugin monitor for each configuration.  You can
  use different custom plugin monitors to monitor different node problems.

#### For Health Checkers

  Health checkers are configured as custom plugins, using the config/health-checker-*.json config files.

#### For Kubernetes exporter

* `--enable-k8s-exporter`: Enables reporting to Kubernetes API server, default to `true`.
* `--apiserver-override`: A URI parameter used to customize how node-problem-detector
connects the apiserver.  This is ignored if `--enable-k8s-exporter` is `false`. The format is the same as the
[`source`](https://github.com/kubernetes/heapster/blob/master/docs/source-configuration.md#kubernetes)
flag of [Heapster](https://github.com/kubernetes/heapster).
For example, to run without auth, use the following config:

   ```
   http://APISERVER_IP:APISERVER_PORT?inClusterConfig=false
   ```

   Refer to [heapster docs](https://github.com/kubernetes/heapster/blob/master/docs/source-configuration.md#kubernetes) for a complete list of available options.
* `--address`: The address to bind the node problem detector server.
* `--port`: The port to bind the node problem detector server. Use 0 to disable.

#### For Prometheus exporter

* `--prometheus-address`: The address to bind the Prometheus scrape endpoint, default to `127.0.0.1`.
* `--prometheus-port`: The port to bind the Prometheus scrape endpoint, default to 20257. Use 0 to disable.

#### For Stackdriver exporter

* `--exporter.stackdriver`: Path to a Stackdriver exporter config file, e.g. [config/exporter/stackdriver-exporter.json](https://github.com/kubernetes/node-problem-detector/blob/master/config/exporter/stackdriver-exporter.json), defaults to empty string. Set to empty string to disable.

### Deprecated Flags

* `--system-log-monitors`: List of paths to system log monitor config files, comma-separated. This option is deprecated, replaced by `--config.system-log-monitor`, and will be removed. NPD will panic if both `--system-log-monitors` and `--config.system-log-monitor` are set.

* `--custom-plugin-monitors`: List of paths to custom plugin monitor config files, comma-separated. This option is deprecated, replaced by `--config.custom-plugin-monitor`, and will be removed. NPD will panic if both `--custom-plugin-monitors` and `--config.custom-plugin-monitor` are set.

## Build Image

* Install development dependencies for `libsystemd` and the ARM GCC toolchain
  * Debian/Ubuntu: `apt install libsystemd-dev gcc-aarch64-linux-gnu`

* `git clone git@github.com:kubernetes/node-problem-detector.git`

* Run `make` in the top directory. It will:
  * Build the binary.
  * Build the container image. The binary and `config/` are copied into the container image.

If you do not need certain categories of problem daemons, you could choose to disable them at compilation time. This is the
best way of keeping your node-problem-detector runtime compact without unnecessary code (e.g. global
variables, goroutines, etc). You can do so via setting the `BUILD_TAGS` environment variable
before running `make`. For example:

`BUILD_TAGS="disable_custom_plugin_monitor disable_system_stats_monitor" make`

The above command will compile the node-problem-detector without [Custom Plugin Monitor](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/custompluginmonitor)
and [System Stats Monitor](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/systemstatsmonitor).
Check out the [Problem Daemon](https://github.com/kubernetes/node-problem-detector#problem-daemon) section
to see how to disable each problem daemon during compilation time.

## Push Image

`make push` uploads the container image to a registry. By default, the image will be uploaded to
`staging-k8s.gcr.io`. It's easy to modify the `Makefile` to push the image
to another registry.

## Installation

The easiest way to install node-problem-detector into your cluster is to use the [Helm](https://helm.sh/) [chart](https://github.com/deliveryhero/helm-charts/tree/master/stable/node-problem-detector):

```
helm repo add deliveryhero https://charts.deliveryhero.io/
helm install --generate-name deliveryhero/node-problem-detector
```

Alternatively, to install node-problem-detector manually:

1. Edit [node-problem-detector.yaml](deployment/node-problem-detector.yaml) to fit your environment. Set `log` volume to your system log directory (used by SystemLogMonitor). You can use a ConfigMap to overwrite the `config` directory inside the pod.

2. Edit [node-problem-detector-config.yaml](deployment/node-problem-detector-config.yaml) to configure node-problem-detector.

3. Edit [rbac.yaml](deployment/rbac.yaml) to fit your environment.

4. Create the ServiceAccount and ClusterRoleBinding with `kubectl create -f rbac.yaml`.

4. Create the ConfigMap with `kubectl create -f node-problem-detector-config.yaml`.

5. Create the DaemonSet with `kubectl create -f node-problem-detector.yaml`.

## Start Standalone

To run node-problem-detector standalone, you should set `inClusterConfig` to `false` and
teach node-problem-detector how to access apiserver with `apiserver-override`.

To run node-problem-detector standalone with an insecure apiserver connection:

```
node-problem-detector --apiserver-override=http://APISERVER_IP:APISERVER_INSECURE_PORT?inClusterConfig=false
```

For more scenarios, see [here](https://github.com/kubernetes/heapster/blob/master/docs/source-configuration.md#kubernetes)

## Windows

Node Problem Detector has preliminary support Windows. Most of the functionality has not been tested but filelog plugin works.

Follow [Issue #461](https://github.com/kubernetes/node-problem-detector/issues/461) for development status of Windows support.

### Development

To develop NPD on Windows you'll need to setup your Windows machine for Go development. Install the following tools:

* [Git for Windows](https://git-scm.com/)
* [Go](https://golang.org/)
* [Visual Studio Code](https://code.visualstudio.com/)
* [Make](http://gnuwin32.sourceforge.net/packages/make.htm)
* [mingw-64 WinBuilds](http://mingw-w64.org/downloads)
  * Tested with x86-64 Windows Native mode.
  * Add the `$InstallDir\bin` to [Windows `PATH` variable](https://answers.microsoft.com/en-us/windows/forum/windows_10-other_settings-winpc/adding-path-variable/97300613-20cb-4d85-8d0e-cc9d3549ba23).

```powershell
# Run these commands in the node-problem-detector directory.

# Build in MINGW64 Window
make clean ENABLE_JOURNALD=0 build-binaries

# Test in MINGW64 Window
make test

# Run with containerd log monitoring enabled in Command Prompt. (Assumes containerd is installed.)
%CD%\output\windows_amd64\bin\node-problem-detector.exe --logtostderr --enable-k8s-exporter=false --config.system-log-monitor=%CD%\config\windows-containerd-monitor-filelog.json --config.system-stats-monitor=config\windows-system-stats-monitor.json

# Configure NPD to run as a Windows Service
sc.exe create NodeProblemDetector binpath= "%CD%\node-problem-detector.exe [FLAGS]" start= demand
sc.exe failure NodeProblemDetector reset= 0 actions= restart/10000
sc.exe start NodeProblemDetector
```

## Try It Out

You can try node-problem-detector in a running cluster by injecting messages to the logs that node-problem-detector is watching. For example, Let's assume node-problem-detector is using [KernelMonitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json). On your workstation, run ```kubectl get events -w```. On the node, run ```sudo sh -c "echo 'kernel: BUG: unable to handle kernel NULL pointer dereference at TESTING' >> /dev/kmsg"```. Then you should see the ```KernelOops``` event.

When adding new rules or developing node-problem-detector, it is probably easier to test it on the local workstation in the standalone mode. For the API server, an easy way is to use ```kubectl proxy``` to make a running cluster's API server available locally. You will get some errors because your local workstation is not recognized by the API server. But you should still be able to test your new rules regardless.

For example, to test [KernelMonitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json) rules:

1. ```make``` (build node-problem-detector locally)
2. ```kubectl proxy --port=8080``` (make a running cluster's API server available locally)
3. Update [KernelMonitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json)'s ```logPath``` to your local kernel log directory. For example, on some Linux systems, it is ```/run/log/journal``` instead of ```/var/log/journal```.
3. ```./bin/node-problem-detector --logtostderr --apiserver-override=http://127.0.0.1:8080?inClusterConfig=false --config.system-log-monitor=config/kernel-monitor.json --config.system-stats-monitor=config/system-stats-monitor.json --port=20256 --prometheus-port=20257``` (or point to any API server address:port and Prometheus port)
4. ```sudo sh -c "echo 'kernel: BUG: unable to handle kernel NULL pointer dereference at TESTING' >> /dev/kmsg"```
5. You can see ```KernelOops``` event in the node-problem-detector log.
6. ```sudo sh -c "echo 'kernel: INFO: task docker:20744 blocked for more than 120 seconds.' >> /dev/kmsg"```
7. You can see ```DockerHung``` event and condition in the node-problem-detector log.
8. You can see ```DockerHung``` condition at [http://127.0.0.1:20256/conditions](http://127.0.0.1:20256/conditions).
9. You can see disk-related system metrics in Prometheus format at [http://127.0.0.1:20257/metrics](http://127.0.0.1:20257/metrics).

**Note**:

* You can see more rule examples under [test/kernel_log_generator/problems](https://github.com/kubernetes/node-problem-detector/tree/master/test/kernel_log_generator/problems).
* For [KernelMonitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json) message injection, all messages should have ```kernel:``` prefix (also note there is a space after ```:```); or use [generator.sh](https://github.com/kubernetes/node-problem-detector/blob/master/test/kernel_log_generator/generator.sh).
* To inject other logs into journald like systemd logs, use ```echo 'Some systemd message' | systemd-cat -t systemd```.

## Dependency Management

node-problem-detector uses [go modules](https://github.com/golang/go/wiki/Modules)
to manage dependencies. Therefore, building node-problem-detector requires
golang 1.11+. It still uses vendoring. See the
[Kubernetes go modules KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/917-go-modules#alternatives-to-vendoring-using-go-modules)
for the design decisions. To add a new dependency, update [go.mod](go.mod) and
run `go mod vendor`.

# Remedy Systems

A _remedy system_ is a process or processes designed to attempt to remedy problems
detected by the node-problem-detector. Remedy systems observe events and/or node
conditions emitted by the node-problem-detector and take action to return the
Kubernetes cluster to a healthy state. The following remedy systems exist:

* [**Descheduler**](https://github.com/kubernetes-sigs/descheduler) strategy RemovePodsViolatingNodeTaints
  evicts pods violating NoSchedule taints on nodes. The k8s scheduler's TaintNodesByCondition feature must
  be enabled. The [Cluster Autoscaler](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler)
  can be used to automatically terminate drained nodes.
* [**mediK8S**](https://github.com/medik8s) is an umbrella project for automatic remediation
  system build on [Node Health Check Operator (NHC)](https://github.com/medik8s/node-healthcheck-operator) that monitors
  node conditions and delegates remediation to external remediators using the Remediation API.[Poison-Pill](https://github.com/medik8s/poison-pill)
  is a remediator that will reboot the node and make sure all statefull workloads are rescheduled. NHC supports conditionally remediating if the cluster
  has enough healthy capacity, or manually pausing any action to minimze cluster disruption.
* [**MachineHealthCheck**](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/machine-health-check) of [Cluster API](https://cluster-api.sigs.k8s.io/) are responsible for remediating unhealthy Machines.

# Testing

NPD is tested via unit tests, [NPD e2e tests](https://github.com/kubernetes/node-problem-detector/blob/master/test/e2e/README.md), Kubernetes e2e tests and Kubernetes nodes e2e tests. Prow handles the [pre-submit tests](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/node-problem-detector/node-problem-detector-presubmits.yaml) and [CI tests](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/node-problem-detector/node-problem-detector-ci.yaml).

CI test results can be found below:

1. [Unit tests](https://testgrid.k8s.io/sig-node-node-problem-detector#ci-npd-test)
2. [NPD e2e tests](https://testgrid.k8s.io/sig-node-node-problem-detector#ci-npd-e2e-test)
3. [Kubernetes e2e tests](https://testgrid.k8s.io/sig-node-node-problem-detector#ci-npd-e2e-kubernetes-gce-gci)
4. [Kubernetes nodes e2e tests](https://testgrid.k8s.io/sig-node-node-problem-detector#ci-npd-e2e-node)

## Running tests

Unit tests are run via `make test`.

See [NPD e2e test documentation](https://github.com/kubernetes/node-problem-detector/blob/master/test/e2e/README.md) for how to set up and run NPD e2e tests.

## Problem Maker

[Problem maker](https://github.com/kubernetes/node-problem-detector/blob/master/test/e2e/problemmaker/README.md) is a program used in NPD e2e tests to generate/simulate node problems. It is ONLY intended to be used by NPD e2e tests. Please do NOT run it on your workstation, as it could cause real node problems.

# Compatibility

Node problem detector's architecture has been fairly stable. Recent versions (v0.8.13+) should be able to work with any supported kubernetes versions.

# Docs

* [Custom plugin monitor](docs/custom_plugin_monitor.md)

# Links

* [Design Doc](https://docs.google.com/document/d/1cs1kqLziG-Ww145yN6vvlKguPbQQ0psrSBnEqpy0pzE/edit?usp=sharing)
* [Slides](https://docs.google.com/presentation/d/1bkJibjwWXy8YnB5fna6p-Ltiy-N5p01zUsA22wCNkXA/edit?usp=sharing)
* [Plugin Interface Proposal](https://docs.google.com/document/d/1jK_5YloSYtboj-DtfjmYKxfNnUxCAvohLnsH5aGCAYQ/edit#)
* [Addon Manifest](https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/node-problem-detector)
* [Metrics Mode Proposal](https://docs.google.com/document/d/1SeaUz6kBavI283Dq8GBpoEUDrHA2a795xtw0OvjM568/edit)
