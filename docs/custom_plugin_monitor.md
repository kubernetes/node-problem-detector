# Custom Plugin Monitor

## Configuration
### Plugin Config
* `invoke_interval`: Interval at which custom plugins will be invoked.
* `timeout`: Time after which custom plugins invokation will be terminated and considered timeout.
* `max_output_length`: The maximum standard output size from custom plugins that NPD will be cut and use for condition status message.
* `concurrency`: The plugin worker number, i.e., how many custom plugins will be invoked concurrently.
* `enable_message_change_based_condition_update`: Flag controls whether message change should result in a condition update.