
# Kubernetes Support for CFS Burst

Automatically configure CFS Burst in kubernetes containers by monitoring namespaces and pods and updating the 
corresponding container directories in /sys/fs/cgroup/cpu,cpuacct/kubepods.

The value of cpu.cfs_burst_us is calculated as a percentage of cpu.cfs_quota_us for each container. The percentage itself
is saved as an annotation in the namespace of the containers.

Example:
```
# namespace annotation
apiVersion: v1
kind: Namespace
metadata:
   name: "test-namespace"
   annotations:
       "cfs.io/burst.percent" : "50"

# configures the containers of ns:test-namespace with
#      cpu.cfs_burst_us = 50% of cpu.cfs_quota_us
```

## Implementation

Implemented as a daemonset that runs on all the nodes of the cluster and has access to the node's file system:

  - The service monitors all the namespaces of the cluster and save a map of the burst percentage associated with each namespace.
  - The service monitors all pods running on it's node for additions/updates, find the corresponding /sys/fs/cgroup/cpu,cpuacct/kubepods/ container directories of the pod and update the value of  cpu.cfs_burst_us.





