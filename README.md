# node-problem-detector
node-problem-detector aims to make various node problems visible to the upstream
layers in cluster management stack.
It is a daemon runs on each node, detects node
problems and reports them to apiserver.
node-problem-detector can either run as a
[DaemonSet](http://kubernetes.io/docs/admin/daemons/) or run standalone on bare metals.
Now it is running as a
[Kubernetes Addon](https://github.com/kubernetes/kubernetes/tree/master/cluster/addons)
enabled by default in the GCE cluster.

# Background
There are tons of node problems could possibly affect the pods running on the
node such as:
* Hardware issues: Bad cpu, memory or disk;
* Kernel issues: Kernel deadlock, corrupted file system;
* Container runtime issues: Unresponsive runtime daemon;
* ...

Currently these problems are invisible to the upstream layers in cluster management
stack, so Kubernetes will continue scheduling pods to the bad nodes.

To solve this problem, we introduced this new daemon **node-problem-detector** to
collect node problems from various daemons and make them visible to the upstream
layers. Once upstream layers have the visibility to those problems, we can discuss the
remedy system.

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

List of supported problem daemons:

| Problem Daemon |  NodeCondition  | Description |
|----------------|:---------------:|:------------|
| [KernelMonitor](https://github.com/kubernetes/node-problem-detector/tree/master/pkg/kernelmonitor) | KernelDeadlock | A problem daemon monitors kernel log and reports problem according to predefined rules. |

# Usage
## Override Apiserver Client Configuration
* `-apiserver-override`  
`apiserver-override` command line argument can customize how to generate a Kubernetes ApiServer
client according to `inClusterConfig` URI parameter. It is the same as
[`Heapster`](https://github.com/kubernetes/heapster.git)'s
[`source` argument](https://github.com/kubernetes/heapster/blob/master/docs/source-configuration.md#kubernetes).
The format is `http://APISERVER_IP:APISERVER_PORT?inClusterConfig=false&userServiceAccount=false&auth=&insecure=`.

## Build Image
Run `make` in the top directory. It will:
* Build the binary.
* Build the docker image. The binary and `config/` are copied into the docker image.
* Upload the docker image to registry. By default, the image will be uploaded to
`gcr.io/google_containers`. It's easy to modify the `Makefile` to push the image
to another registry

## Start DaemonSet
* Create a file node-problem-daemon.yaml with the following yaml.
```yaml
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: node-problem-detector
spec:
  template:
    spec:
      containers:
      - name: node-problem-detector
        image: gcr.io/google_containers/node-problem-detector:v0.2
        imagePullPolicy: Always
        securityContext:
          privileged: true
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - name: log
          mountPath: /log
          readOnly: true
      volumes:
      - name: log
        # Config `log` to your system log directory
        hostPath:
          path: /var/log/
```
* Edit node-problem-detector.yaml to fit your environment: Set `log` volume to your system log diretory. (Used by KernelMonitor)
* Create the DaemonSet with `kubectl create -f node-problem-detector.yaml`
* If needed, you can use [ConfigMap](http://kubernetes.io/docs/user-guide/configmap/)
to overwrite the `config/`.

## Start Standalone
`inClusterConfig` should be set to `false`. To run node-problem-detector standalone with an insecure apiserver connection: 
`node-problem-detector -apiserver-override=http://APISERVER_IP:APISERVER_PORT?inClusterConfig=false`

# Links
* [Design Doc](https://docs.google.com/document/d/1cs1kqLziG-Ww145yN6vvlKguPbQQ0psrSBnEqpy0pzE/edit?usp=sharing)
* [Slides](https://docs.google.com/presentation/d/1bkJibjwWXy8YnB5fna6p-Ltiy-N5p01zUsA22wCNkXA/edit?usp=sharing)
* [Addon Manifest](https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/node-problem-detector)
