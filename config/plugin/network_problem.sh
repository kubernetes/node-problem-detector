#!/bin/bash

# This plugin checks for common network issues. Currently, it only checks
# if the conntrack table is full. 

OK=0
NONOK=1
UNKNOWN=2

[ -f /proc/sys/net/ipv4/netfilter/ip_conntrack_max ] || echo $UNKNOWN
[ -f /proc/sys/net/ipv4/netfilter/ip_conntrack_count ] || echo $UNKNOWN

conntrack_max=$(cat /proc/sys/net/ipv4/netfilter/ip_conntrack_max)
conntrack_count=$(cat /proc/sys/net/ipv4/netfilter/ip_conntrack_count)

if (( conntrack_count >= conntrack_max )); then
  echo "Conntrack table full"
  exit $NONOK
fi

echo "Conntrack table available"
exit $OK

