
# Kubernetes Support for CFS Burst

Automatically configure CFS Burst in kubernetes containers by monitoring namespaces and pods and updating the 
corresponding container directories in /sys/fs/cgroup/cpu,cpuacct/kubepods.

The value of cpu.cfs_burst_us is calculated as a percentage of cpu.cfs_quota_us for each container. The percentage itself
is saved as an annotation in the namespace of the containers.

To use CFS Burst, the feature must be globally enabled on the nodes:

```
echo 1 > /proc/sys/kernel/sched_cfs_bw_burst_enabled

```
In some OS like flatcar, this directory is read-only and can only be updated by creating a different OS image.

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

## Note

It's possible to use cfs_burst_us even when the feature is not globally enabled. In this case, the value of cfs_burst_us cannot exceed cfs_qquota_us but the nr_bursts and nr_burst_time values in the file cpu.stats will still be updated.


## Reference:

https://www.alibabacloud.com/help/en/alinux/user-guide/enable-the-cpu-burst-feature-for-cgroup-v1?source=post_page-----ac7fae302c99--------------------------------


