#!/bin/bash

# This plugin checks for common network issues.
# Currently only checks if conntrack table is more than 90% used.

readonly OK=0
readonly NONOK=1
readonly UNKNOWN=2

# "nf_conntrack" replaces "ip_conntrack" - support both
readonly NF_CT_COUNT_PATH='/proc/sys/net/netfilter/nf_conntrack_count'
readonly NF_CT_MAX_PATH='/proc/sys/net/netfilter/nf_conntrack_max'
readonly IP_CT_COUNT_PATH='/proc/sys/net/ipv4/netfilter/ip_conntrack_count'
readonly IP_CT_MAX_PATH='/proc/sys/net/ipv4/netfilter/ip_conntrack_max'

if [[ -f $NF_CT_COUNT_PATH ]] && [[ -f $NF_CT_MAX_PATH ]]; then
  readonly CT_COUNT_PATH=$NF_CT_COUNT_PATH
  readonly CT_MAX_PATH=$NF_CT_MAX_PATH
elif [[ -f $IP_CT_COUNT_PATH ]] && [[ -f $IP_CT_MAX_PATH ]]; then
  readonly CT_COUNT_PATH=$IP_CT_COUNT_PATH
  readonly CT_MAX_PATH=$IP_CT_MAX_PATH
else
  exit $UNKNOWN
fi

readonly conntrack_count=$(< $CT_COUNT_PATH) || exit $UNKNOWN
readonly conntrack_max=$(< $CT_MAX_PATH) || exit $UNKNOWN
readonly conntrack_usage_msg="${conntrack_count} out of ${conntrack_max}"

if (( conntrack_count > conntrack_max * 9 /10 )); then
  echo "Conntrack table usage over 90%: ${conntrack_usage_msg}"
  exit $NONOK
else
  echo "Conntrack table usage: ${conntrack_usage_msg}"
  exit $OK
fi
