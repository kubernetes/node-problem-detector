#!/bin/bash

# NOTE: THIS NTP SERVICE CHECK SCRIPT ASSUME THAT NTP SERVICE IS RUNNING UNDER SYSTEMD.
#       THIS IS JUST AN EXAMPLE. YOU CAN WRITE YOUR OWN NODE PROBLEM PLUGIN ON DEMAND.

systemctl status ntp.service | grep 'Active:' | grep -q 'running'
ret=$?
if [ $ret -ne 0 ]; then
    echo "NTP service is down."
    exit 1
fi

echo "NTP service is up."
exit 0