# System Stats Monitor

*System Stats Monitor* is a problem daemon in node problem detector. It collects pre-defined health-related metrics from different system components.  Each component may allow further detailed configurations.

Currently supported components are:

* cpu
* disk
* host
* memory

See example config file [here](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json).

By setting the `metricsConfigs` field and `displayName` field ([example](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json)), you can specify the list of metrics to be collected, and their display names on the Prometheus scaping endpoint.

## Detailed Configuration Options

### Global Configurations

Data collection period can be specified globally in the config file, see `invokeInterval` at the [example](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json).

### CPU

Below metrics are collected from `cpu` component:

* `cpu_runnable_task_count`: The average number of runnable tasks in the run-queue during the last minute. Collected from [`/proc/loadavg`][/proc doc].
* `cpu_usage_time`: CPU usage, in seconds. The [CPU state][/proc doc] for the corresponding usage is reported under the `state` metric label (e.g. `user`, `nice`, `system`...).

[/proc doc]: http://man7.org/linux/man-pages/man5/proc.5.html

### Disk

Below metrics are collected from `disk` component:

* `disk_io_time`: [# of milliseconds spent doing I/Os on this device][iostat doc]
* `disk_weighted_io`: [# of milliseconds spent doing I/Os on this device][iostat doc]
* `disk_avg_queue_len`: [average # of requests that was waiting in queue or being serviced during the last `invokeInterval`](https://www.xaprb.com/blog/2010/01/09/how-linux-iostat-computes-its-results/)
* `disk_operation_count`: [# of reads/writes completed][iostat doc]
* `disk_merged_operation_count`: [# of reads/writes merged][iostat doc]
* `disk_operation_bytes_count`: # of Bytes used for reads/writes on this device
* `disk_operation_time`: [# of milliseconds spent reading/writing][iostat doc]
* `disk_bytes_used`: Disk usage in Bytes. The usage state is reported under the `state` metric label (e.g. `used`, `free`). Summing values of all states yields the disk size.

The name of the disk block device is reported in the `device_name` metric label (e.g. `sda`).

For the metrics that separates read/write operations, the IO direction is reported in the `direction` metric label (e.g. `read`, `write`).

And a few other options:
* `includeRootBlk`: When set to `true`, add all block devices that's [not a slave or holder device][lsblk doc] to the list of disks that System Stats Monitor collects metrics from. When set to `false`, do not modify the list of disks that System Stats Monitor collects metrics from.
* `includeAllAttachedBlk`: When set to `true`, add all currently attached block devices to the list of disks that System Stats Monitor collects metrics from. When set to `false`, do not modify the list of disks that System Stats Monitor collects metrics from.
* `lsblkTimeout`: System Stats Monitor uses [`lsblk`][lsblk doc] to retrieve block devices information. This option sets the timeout for calling `lsblk` commands.

[iostat doc]: https://www.kernel.org/doc/Documentation/iostats.txt
[lsblk doc]: http://man7.org/linux/man-pages/man8/lsblk.8.html

### Host

Below metrics are collected from `host` component:

* `host_uptime`: The uptime of the operating system, in seconds. OS version and kernel versions are reported under the `os_version` and `kernel_version` metric label (e.g. `cos 73-11647.217.0`, `4.14.127+`).

### Memory

Below metrics are collected from `memory` component:

* `memory_bytes_used`: Memory usage by each memory state, in Bytes. The memory state is reported under the `state` metric label (e.g. `free`, `used`, `buffered`...). Summing values of all states yields the total memory of the node.
* `memory_anonymous_used`: Anonymous memory usage, in Bytes. Memory usage state is reported under the `state` metric label (e.g. `active`, `inactive`). `active` means the memory has been used more recently and usually not swapped until needed. Summing values of all states yields the total anonymous memory used.
* `memory_page_cache_used`: Page cache memory usage, in Bytes. Memory usage state is reported under the `state` metric label (e.g. `active`, `inactive`). `active` means the memory has been used more recently and usually not reclaimed until needed. Summing values of all states yields the total page cache memory used.
* `memory_unevictable_used`: [Unevictable memory][/proc doc] usage, in Bytes.
* `memory_dirty_used`: Dirty pages usage, in Bytes. Memory usage state is reported under the `state` metric label (e.g. `dirty`, `writeback`). `dirty` means the memory is waiting to be written back to disk, and `writeback` means the memory is actively being written back to disk.
