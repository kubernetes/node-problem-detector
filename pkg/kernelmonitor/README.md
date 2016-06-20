# Kernel Monitor

*Kernel Monitor* is a problem daemon in node problem detector. It monitors kernel log
and detects known kernel issues following predefined rules.

The Kernel Monitor matches kernel issues according to a set of predefined rule list in
[`config/kernel-monitor.json`](https://github.com/kubernetes/node-problem-detector/blob/master/config/kernel-monitor.json).
The rule list is extensible.

## Limitations

* Kernel Monitor only supports file based kernel log now. It doesn't support log tools
like journald. There is an [open issue](https://github.com/kubernetes/node-problem-detector/issues/14)
to add journald support.

* Kernel Monitor has assumption on kernel log format, now it only works on Ubuntu and
Debian. However, it is easy to extend it to [support other log format](#support-other-log-format).

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

## Change Log Path

Kernel log in different OS distros may locate in different path. The `log`
field in `config/kernel-monitor.json` is the log path inside the container.
You can always configure it to match your OS distro.

## Support Other Log Format

Kernel monitor uses [`Translator`](https://github.com/kubernetes/node-problem-detector/blob/master/pkg/kernelmonitor/translator/translator.go)
plugin to translate kernel log the internal data structure. It is easy to
implement a new translator for a new log format.
