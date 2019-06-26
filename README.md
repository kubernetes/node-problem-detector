# node-problem-detector

[![Build Status](https://travis-ci.org/kubernetes/node-problem-detector.svg?branch=master)](https://travis-ci.org/kubernetes/node-problem-detector)  [![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes/node-problem-detector)](https://goreportcard.com/report/github.com/kubernetes/node-problem-detector)

node-problem-detector aims to make various node problems visible to the upstream
layers in cluster management stack.
It is a daemon which runs on each node, detects node
problems and reports them to apiserver.
node-problem-detector can either run as a
[DaemonSet](http://kubernetes.io/docs/admin/daemons/) or run standalone.
Now it is running as a
[Kubernetes Addon](https://github.com/kubernetes/kubernetes/tree/master/cluster/addons)
enabled by default in the GCE cluster.

# Background

There are tons of node problems could possibly affect the pods running on the
node such as:
* Infrastructure daemon issues: ntp service down;
* Hardware issues: Bad cpu, memory or disk, ntp service down;
* Kernel issues: Kernel deadlock, corrupted file system;
* Container runtime issues: Unresponsive runtime daemon;
* ...

Currently these problems are invisible to the upstream layers in cluster management
stack, so Kubernetes will continue scheduling pods to the bad nodes.

To solve this problem, we introduced this new daemon **node-problem-detector** to
collect node problems from various daemons and make them visible to the upstream
layers. Once upstream layers have the visibility to those problems, we can discuss the
[remedy system](#remedy-systems).

# Problem API

node-problem-detector uses `Event` and `NodeCondition` to report problems to
apiserver.
* `NodeCondition`: Permanent problem that makes the node unavailable for pods should
be reported as `NodeCondition`.
* `Event`: Temporary problem that has limited impact on pod but is informative
should be reported as `Event`.

# Problem Daemon

A problem daemon is a sub-daemon of node-problem-detector. It monitors a specific
kind of node problems and reports them to node-problem-detector.

A problem daemon could be:
* A tiny daemon designed for dedicated usecase of Kubernetes.
* An existing node health monitoring daemon integrated with node-problem-detector.

Currently, a problem daemon is running as a goroutine in the node-problem-detector
binary. In the future, we'll separate node-problem-detector and problem daemons into
different containers, and compose them with pod specification.

Each catagory of problem daemon can be disabled at compilation time by setting
corresponding build tags. If they are disabled at compilation time, then all their
build dependencies, global variables and background goroutines will be trimmed out
of the compiled executable.

List of supported problem daemons:

| Problem Daemon |  NodeCondition  | Description | Disabling Build Tag |
|----------------|:---------------:|:------------|:--------------------|
| [KernelMonitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json) | KernelDeadlock | A system log monitor monitors kernel log and reports problems and metrics according to predefined rules. | disable_system_log_monitor
| [AbrtAdaptor](https://github.com/kubernetes/node-problem-detector/blob/master/config/abrt-adaptor.json) | None | Monitor ABRT log messages and report them further. ABRT (Automatic Bug Report Tool) is health monitoring daemon able to catch kernel problems as well as application crashes of various kinds occurred on the host. For more information visit the [link](https://github.com/abrt). | disable_system_log_monitor
| [CustomPluginMonitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/custom-plugin-monitor.json) | On-demand(According to users configuration) | A custom plugin monitor for node-problem-detector to invoke and check various node problems with user defined check scripts. See proposal [here](https://docs.google.com/document/d/1jK_5YloSYtboj-DtfjmYKxfNnUxCAvohLnsH5aGCAYQ/edit#). | disable_custom_plugin_monitor
| [SystemStatsMonitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json) | None(Could be added in the future) | A system stats monitor for node-problem-detector to collect various health-related system stats as metrics. See proposal [here](https://docs.google.com/document/d/1SeaUz6kBavI283Dq8GBpoEUDrHA2a795xtw0OvjM568/edit). | disable_system_stats_monitor

# Exporter

An exporter is a component of node-problem-detector. It reports node problems and/or metrics to 
certain back end (e.g. Kubernetes API server, or Prometheus scrape endpoint).

# Usage

## Flags

* `--version`: Print current version of node-problem-detector.
* `--address`: The address to bind the node problem detector server.
* `--port`: The port to bind the node problem detector server. Use 0 to disable.
* `--config.system-log-monitor`: List of paths to system log monitor configuration files, comma separated, e.g.
  [config/kernel-monitor.json](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json).
  Node problem detector will start a separate log monitor for each configuration. You can
  use different log monitors to monitor different system log.
* `--config.custom-plugin-monitor`: List of paths to custom plugin monitor config files, comma separated, e.g.
  [config/custom-plugin-monitor.json](https://github.com/kubernetes/node-problem-detector/blob/master/config/custom-plugin-monitor.json).
  Node problem detector will start a separate custom plugin monitor for each configuration. You can
  use different custom plugin monitors to monitor different node problems.
* `--config.system-stats-monitor`: List of paths to system stats monitor config files, comma separated, e.g.
  [config/system-stats-monitor.json](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json).
  Node problem detector will start a separate system stats monitor for each configuration. You can
  use different system stats monitors to monitor different problem-related system stats.
* `--enable-k8s-exporter`: Enables reporting to Kubernetes API server, default to `true`.
* `--apiserver-override`: A URI parameter used to customize how node-problem-detector
connects the apiserver.  This is ignored if `--enable-k8s-exporter` is `false`. The format is same as the
[`source`](https://github.com/kubernetes/heapster/blob/master/docs/source-configuration.md#kubernetes)
flag of [Heapster](https://github.com/kubernetes/heapster).
For example, to run without auth, use the following config:
   ```
   http://APISERVER_IP:APISERVER_PORT?inClusterConfig=false
   ```
   Refer [heapster docs](https://github.com/kubernetes/heapster/blob/master/docs/source-configuration.md#kubernetes) for a complete list of available options.
* `--hostname-override`: A customized node name used for node-problem-detector to update conditions and emit events. node-problem-detector gets node name first from `hostname-override`, then `NODE_NAME` environment variable and finally fall back to `os.Hostname`.
* `--prometheus-address`: The address to bind the Prometheus scrape endpoint, default to `127.0.0.1`.
* `--prometheus-port`: The port to bind the Prometheus scrape endpoint, default to 20257. Use 0 to disable.

### Deprecated Flags

* `--system-log-monitors`: List of paths to system log monitor config files, comma separated. This option is deprecated, replaced by `--config.system-log-monitor`, and will be removed. NPD will panic if both `--system-log-monitors` and `--config.system-log-monitor` are set.

* `--custom-plugin-monitors`: List of paths to custom plugin monitor config files, comma separated. This option is deprecated, replaced by `--config.custom-plugin-monitor`, and will be removed. NPD will panic if both `--custom-plugin-monitors` and `--config.custom-plugin-monitor` are set.

## Build Image

* `go get` or `git clone` node-problem-detector repo into `$GOPATH/src/k8s.io` or `$GOROOT/src/k8s.io`
with one of the below directions:
  * `cd $GOPATH/src/k8s.io && git clone git@github.com:kubernetes/node-problem-detector.git`
  * `cd $GOROOT/src/k8s.io && git clone git@github.com:kubernetes/node-problem-detector.git`
  * `cd $GOPATH/src/k8s.io && go get k8s.io/node-problem-detector`

* run `make` in the top directory. It will:
  * Build the binary.
  * Build the docker image. The binary and `config/` are copied into the docker image.

If you do not need certain categories of problem daemons, you could choose to disable them at compilation time. This is the
best way of keeping your node-problem-detector runtime compact without unnecessary code (e.g. global
variables, goroutines, etc). You can do so via setting the `BUILD_TAGS` environment variable
before running `make`. For example:

`BUILD_TAGS="disable_custom_plugin_monitor disable_system_stats_monitor" make`

Above command will compile the node-problem-detector without [Custom Plugin Monitor](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/custompluginmonitor)
and [System Stats Monitor](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/systemstatsmonitor).
Check out the [Problem Daemon](https://github.com/kubernetes/node-problem-detector#problem-daemon) section
to see how to disable each problem daemon during compilation time.

**Note**:
By default node-problem-detector will be built with systemd support with `make` command. This requires systemd develop files.
You should download the systemd develop files first. For Ubuntu, `libsystemd-journal-dev` package should
be installed. For Debian, `libsystemd-dev` package should be installed.

## Push Image

`make push` uploads the docker image to registry. By default, the image will be uploaded to
`staging-k8s.gcr.io`. It's easy to modify the `Makefile` to push the image
to another registry.

## Installation

The easiest way to install node-problem-detector into your cluster is to use the [Helm](https://helm.sh/) [chart](https://github.com/helm/charts/tree/master/stable/node-problem-detector):

```
helm install stable/node-problem-detector
```

Or alternatively, to install node-problem-detector manually:

1. Edit [node-problem-detector.yaml](deployment/node-problem-detector.yaml) to fit your environment. Set `log` volume to your system log directory (used by SystemLogMonitor). You can use a ConfigMap to overwrite the `config` directory inside the pod. For Kubernetes versions older than 1.9, use [node-problem-detector-old.yaml](deployment/node-problem-detector-old.yaml).

2. Edit [node-problem-detector-config.yaml](deployment/node-problem-detector-config.yaml) to configure node-problem-detector.

3. Create the ConfigMap with `kubectl create -f node-problem-detector-config.yaml`.

3. Create the DaemonSet with `kubectl create -f node-problem-detector.yaml`.

## Start Standalone

To run node-problem-detector standalone, you should set `inClusterConfig` to `false` and
teach node-problem-detector how to access apiserver with `apiserver-override`.

To run node-problem-detector standalone with an insecure apiserver connection:
```
node-problem-detector --apiserver-override=http://APISERVER_IP:APISERVER_INSECURE_PORT?inClusterConfig=false
```

For more scenarios, see [here](https://github.com/kubernetes/heapster/blob/master/docs/source-configuration.md#kubernetes)

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
9. You can see disk related system metrics in Prometheus format at [http://127.0.0.1:20257/metrics](http://127.0.0.1:20257/metrics).

**Note**:
- You can see more rule examples under [test/kernel_log_generator/problems](https://github.com/kubernetes/node-problem-detector/tree/master/test/kernel_log_generator/problems).
- For [KernelMonitor](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json) message injection, all messages should have ```kernel: ``` prefix (also note there is a space after ```:```); or use [generator.sh](https://github.com/kubernetes/node-problem-detector/blob/master/test/kernel_log_generator/generator.sh).
- To inject other logs into journald like systemd logs, use ```echo 'Some systemd message' | systemd-cat -t systemd```.

## Dependency Management

node-problem-detector uses [go modules](https://github.com/golang/go/wiki/Modules)
to manage dependencies. Therefore, building node-problem-detector requires
golang 1.11+. It still uses vendoring. See the
[Kubernetes go modules KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-architecture/2019-03-19-go-modules.md#alternatives-to-vendoring-using-go-modules)
for the design decisions. To add a new dependency, update [go.mod](go.mod) and
run `GO111MODULE=on go mod vendor`.

# Remedy Systems

A _remedy system_ is a process or processes designed to attempt to remedy problems
detected by the node-problem-detector. Remedy systems observe events and/or node
conditions emitted by the node-problem-detector and take action to return the
Kubernetes cluster to a healthy state. The following remedy systems exist:

* [**Draino**](https://github.com/planetlabs/draino) automatically drains Kubernetes
  nodes based on labels and node conditions. Nodes that match _all_ of the supplied
  labels and _any_ of the supplied node conditions will be prevented from accepting
  new pods (aka 'cordoned') immediately, and
  [drained](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)
  after a configurable time. Draino can be used in conjunction with the
  [Cluster Autoscaler](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler)
  to automatically terminate drained nodes. Refer to
  [this issue](https://github.com/kubernetes/node-problem-detector/issues/199)
  for an example production use case for Draino.

# Docs

* [Custom plugin monitor](docs/custom_plugin_monitor.md)

# Links

* [Design Doc](https://docs.google.com/document/d/1cs1kqLziG-Ww145yN6vvlKguPbQQ0psrSBnEqpy0pzE/edit?usp=sharing)
* [Slides](https://docs.google.com/presentation/d/1bkJibjwWXy8YnB5fna6p-Ltiy-N5p01zUsA22wCNkXA/edit?usp=sharing)
* [Plugin Interface Proposal](https://docs.google.com/document/d/1jK_5YloSYtboj-DtfjmYKxfNnUxCAvohLnsH5aGCAYQ/edit#)
* [Addon Manifest](https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/node-problem-detector)
* [Metrics Mode Proposal](https://docs.google.com/document/d/1SeaUz6kBavI283Dq8GBpoEUDrHA2a795xtw0OvjM568/edit)
