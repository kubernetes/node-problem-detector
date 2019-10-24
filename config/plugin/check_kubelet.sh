#!/bin/bash

# NOTE: THIS NTP SERVICE CHECK SCRIPT ASSUME THAT NTP SERVICE IS RUNNING UNDER SYSTEMD.
#       THIS IS JUST AN EXAMPLE. YOU CAN WRITE YOUR OWN NODE PROBLEM PLUGIN ON DEMAND.

OK=0
NONOK=1
UNKNOWN=2

which systemctl >/dev/null
if [ $? -ne 0 ]; then
    echo "Systemd is not supported"
    exit $UNKNOWN
fi

systemctl status kubelet.service | grep 'Active:' | grep -q running
if [ $? -ne 0 ]; then
    echo "kubelet service is not running"
    exit $NONOK
fi

echo "kubelet service is running"
exit $OK
