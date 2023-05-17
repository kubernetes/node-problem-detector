# Custom Plugin Monitor

## Configuration

### Plugin Config

* `invoke_interval`: Interval at which custom plugins will be invoked.
* `timeout`: Time after which custom plugins invocation will be terminated and considered timeout.
* `max_output_length`: The maximum standard output size from custom plugins that NPD will be cut and use for condition status message.
* `concurrency`: The plugin worker number, i.e., how many custom plugins will be invoked concurrently.
* `enable_message_change_based_condition_update`: Flag controls whether message change should result in a condition update.
* `skip_initial_status`: Flag controls whether condition will be emitted during plugin initialization.

### Annotated Plugin Configuration Example

```
{
  "plugin": "custom",
  "pluginConfig": {
    "invoke_interval": "30s",
    "timeout": "5s",
    "max_output_length": 80,
    "concurrency": 3,
    "enable_message_change_based_condition_update": false
  },
  "source": "ntp-custom-plugin-monitor",
  "metricsReporting": true,
  "conditions": [
    {
      "type": "NTPProblem",
      "reason": "NTPIsUp",              // This is the default reason shown when healthy
      "message": "ntp service is up"    // This is the default message shown when healthy
    }
  ],
  "rules": [
    {
      "type": "temporary",              // These are not shown unless there's an
                                        // event so they always relate to a problem.
                                        // There are no defaults since there is nothing
                                        // to show unless there's a problem.
      "reason": "NTPIsDown",            // This is the reason shown for this event
                                        // and the message shown comes from stdout.
      "path": "./config/plugin/check_ntp.sh",
      "timeout": "3s"
    },
    {
      "type": "permanent",              // These are permanent and are shown in the Conditions section
                                        // when running `kubectl describe node ...`
                                        // They have default values shown above in the conditions section
                                        // and also a reason for each specific trigger listed in this rules section.
                                        // Message will come from default for healthy times
                                        // and during unhealthy time message comes from stdout of the check.
 
      "condition": "NTPProblem",        // This is the key to connect to the corresponding condition listed above
      "reason": "NTPIsDown",            // and the reason shown for failures detected in this rule
                                        // and message will be from stdout of the check.
      "path": "./config/plugin/check_ntp.sh",
      "timeout": "3s"
    }
  ]
}
```
