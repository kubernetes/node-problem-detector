# Pod Restart Monitor

*Pod Restart Monitor* is a problem daemon in node problem detector. It monitors the number of times a pod has been
restarted.

The original use case was for monitoring the `aws-node` DaemonSet in EKS clusters. Multiple restarts of that pod can
indicate a hard to define, but definite underlying condition on a node that makes it unsuitable for scheduling.

### Plugin Configuration

The plugin checks a very simple condition: how many times any container in a pod has restarted in its lifetime. 
This condition will not go away unless the pod is deleted.

The defaults are tuned for the aws-node daemon in EKS clusters, but any pod can be monitored:

/config/aws-restart-monitor.json:

```json
{
  "plugin": "podrestart",
  "source": "pod-restart-monitor",
  "namespace": "kube-system",
  "podSelector": "k8s-app=aws-node",
  "restartThreshold": 5,
  "checkInterval": "5m"
}
```

### RBAC Requirements

The plugin needs to be able to get pods and nodes, which can be accomplished with an additional ClusterRole:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: get-pods-and-nodes
rules:
  - verbs:
      - get
      - list
      - watch
    apiGroups:
      - ""
    resources:
      - pods
      - nodes

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: node-problem-detector-get-pods-and-nodes
subjects:
  - kind: ServiceAccount
    name: node-problem-detector
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: npd-list-pods
  apiGroup: rbac.authorization.k8s.io
```
