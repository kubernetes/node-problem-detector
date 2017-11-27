#!/bin/bash

# This plugin checks for common network issues. Currently, it only checks
# if the conntrack table is full. 

conntrack_max=$(cat /proc/sys/net/ipv4/netfilter/ip_conntrack_max)
conntrack_count=$(cat /proc/sys/net/ipv4/netfilter/ip_conntrack_count)
if (( conntrack_count >= conntrack_max )); then
  echo "Conntrack table full"
  exit 1
fi

echo "Conntrack table available"
exit 0

