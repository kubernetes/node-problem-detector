#!/bin/bash

# This plugin checks if the ntp service is running under systemd.
# NOTE: This is only an example for systemd services.

readonly OK=0
readonly NONOK=1
readonly UNKNOWN=2

readonly SERVICE='ntp.service'

# Check systemd cmd present
if ! command -v systemctl >/dev/null; then
  echo "Could not find 'systemctl' - require systemd"
  exit $UNKNOWN
fi

# Return success if service active (i.e. running)
if systemctl -q is-active "$SERVICE"; then
  echo "$SERVICE is running"
  exit $OK
else
  # Does not differenciate stopped/failed service from non-existent
  echo "$SERVICE is not running"
  exit $NONOK
fi

