# System Stats Monitor

*System Stats Monitor* is a problem daemon in node problem detector. It collects pre-defined health-related metrics from different system components.  Each component may allow further detailed configurations.

Currently supported components are:

* disk

See example config file [here](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json).

## Detailed Configuration Options

### Global Configurations

Data collection period can be specified globally in the config file, see `invokeInterval` at the [example](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json).

### Disk

Below metrics are collected from `disk` component:

* `disk/io_time`: [# of milliseconds spent doing I/Os on this device](https://www.kernel.org/doc/Documentation/iostats.txt)
* `disk/weighted_io`: [# of milliseconds spent doing I/Os on this device](https://www.kernel.org/doc/Documentation/iostats.txt)
* `disk/avg_queue_len`: [average # of requests that was waiting in queue or being serviced during the last `invokeInterval`](https://www.xaprb.com/blog/2010/01/09/how-linux-iostat-computes-its-results/)

By setting the `metricsConfigs` field and `displayName` field ([example](https://github.com/kubernetes/node-problem-detector/blob/master/config/system-stats-monitor.json)), you can specify the list of metrics to be collected, and their display names on the Prometheus scaping endpoint. The name of the disk block device will be reported in the `device` metrics label.

And a few other options:
* `includeRootBlk`: When set to `true`, add all block devices that's [not a slave or holder device](http://man7.org/linux/man-pages/man8/lsblk.8.html) to the list of disks that System Stats Monitor collects metrics from. When set to `false`, do not modify the list of disks that System Stats Monitor collects metrics from.
* `includeAllAttachedBlk`: When set to `true`, add all currently attached block devices to the list of disks that System Stats Monitor collects metrics from. When set to `false`, do not modify the list of disks that System Stats Monitor collects metrics from.
* `lsblkTimeout`: System Stats Monitor uses [`lsblk`](http://man7.org/linux/man-pages/man8/lsblk.8.html) to retrieve block devices information. This option sets the timeout for calling `lsblk` commands.
