# System Log Monitor

*System Log Monitor* is a problem daemon in node problem detector. It monitors
specified system daemon log and detects problems following predefined rules.

The System Log Monitor matches problems according to a set of predefined rule list in
the configuration files. (
[`config/kernel-monitor.json`](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json) as an example).
The rule list is extensible.

## Supported sources

* System Log Monitor currently supports file-based logs, journald, and kmsg.
  Additional sources can be added by implementing a [new log
  watcher](#new-log-watcher).

## Add New NodeConditions

To support new node conditions, you can extend the `conditions` field in
the configuration file with new condition definition:

```json
{
  "type": "NodeConditionType",
  "reason": "CamelCaseDefaultNodeConditionReason",
  "message": "arbitrary default node condition message"
}
```

## Detect New Problems

To detect new problems, you can extend the `rules` field in the configuration file
with new rule definition:

```json
{
  "type": "temporary/permanent",
  "condition": "NodeConditionOfPermanentIssue",
  "reason": "CamelCaseShortReason",
  "message": "regexp matching the issue in the log"
}
```

*Note that the pattern must match to the end of the line excluding the
tailing newline character, and multi-line pattern is supported.*

## Log Watchers

System log monitor supports different log management tools with different log
watchers:
* [filelog](./logwatchers/filelog): Log watcher for
arbitrary file based log.
* [journald](.//logwatchers/journald): Log watcher for journald.
* [kmsg](./logwatchers/kmsg): Log watcher for the kernel ring buffer device, /dev/kmsg.
Set `plugin` in the configuration file to specify log watcher.

### Plugin Configuration

Log watcher specific configurations are configured in `pluginConfig`.
* **journald**
  * source: The [`SYSLOG_IDENTIFIER`](https://www.freedesktop.org/software/systemd/man/systemd.journal-fields.html)
  of the log to watch.
* **filelog**:
  * timestamp: The regular expression used to match timestamp in the log line.
    Submatch is supported, but only the last result will be used as the actual
    timestamp.
  * message: The regular expression used to match message in the log line.
    Submatch is supported, but only the last result will be used as the actual
    message.
  * timestampFormat: The format of the timestamp. The format string is the time
    `2006-01-02T15:04:05Z07:00` in the expected format. (See
    [golang timestamp format](https://golang.org/pkg/time/#pkg-constants))
* **kmsg**: No configuration for now.

### Change Log Path

Log on different OS distros may locate in different path. The `logPath`
field in the configuration file is the log path. You can always configure
`logPath` to match your OS distro.
* filelog: `logPath` is the path of log file, e.g. `/var/log/kern.log` for kernel
  log.
* journald: `logPath` is the journal log directory, usually `/var/log/journal`.

### New Log Watcher

System log monitor uses [Log Watcher](./logwatchers/types/log_watcher.go) to
support different log management tools.  It is easy to implement a new log
watcher.
