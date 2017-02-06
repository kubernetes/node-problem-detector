# Kernel Monitor

*Kernel Monitor* is a problem daemon in node problem detector. It monitors kernel log
and detects known kernel issues following predefined rules.

The Kernel Monitor matches kernel issues according to a set of predefined rule list in
[`config/kernel-monitor.json`](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json).
The rule list is extensible.

## Limitations

* Kernel Monitor only supports syslog (rsyslog) and journald now, but it is easy
  to extend it with [new log watcher](#new-log-watcher)

## Add New NodeConditions

To support new node conditions, you can extend the `conditions` field in
`config/kernel-monitor.json` with new condition definition:

```json
{
  "type": "NodeConditionType",
  "reason": "CamelCaseDefaultNodeConditionReason",
  "message": "arbitrary default node condition message"
}
```

## Detect New Problems

To detect new problems, you can extend the `rules` field in `config/kernel-monitor.json`
with new rule definition:

```json
{
  "type": "temporary/permanent",
  "condition": "NodeConditionOfPermanentIssue",
  "reason": "CamelCaseShortReason",
  "message": "regexp matching the issue in the kernel log"
}
```

## Log Watchers

Kernel monitor supports different log management tools with different log
watchers:
* [syslog](https://github.com/kubernetes/node-problem-detector/blob/master/pkg/logmonitor/logwatchers/syslog)
* [journald](https://github.com/kubernetes/node-problem-detector/blob/master/pkg/logmonitor/logwatchers/journald)

### Change Log Path

Kernel log on different OS distros may locate in different path. The `logPath`
field in `config/kernel-monitor.json` is the log path inside the container.
You can always configure `logPath` and volume mount to match your OS distro.
* syslog: `logPath` is the kernel log path, usually `/var/log/kern.log`.
* journald: `logPath` is the journal log directory, usually `/var/log/journal`.

### New Log Watcher

Kernel monitor uses [Log
Watcher](https://github.com/kubernetes/node-problem-detector/blob/master/pkg/logmonitor/logwatchers/types/log_watcher.go) to support different log management tools.
It is easy to implement a new log watcher.
