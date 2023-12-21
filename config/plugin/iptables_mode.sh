#!/bin/bash

# As of iptables 1.8, the iptables command line clients come in two different versions/modes: "legacy",
# which uses the kernel iptables API just like iptables 1.6 and earlier did, and "nft", which translates
# the iptables command-line API into the kernel nftables API.
# Because they connect to two different subsystems in the kernel, you cannot mix rules from different versions.
# Ref: https://github.com/kubernetes-sigs/iptables-wrappers

readonly OK=0
readonly NONOK=1
readonly UNKNOWN=2

# based on: https://github.com/kubernetes-sigs/iptables-wrappers/blob/97b01f43a8e8db07840fc4b95e833a37c0d36b12/iptables-wrapper-installer.sh
readonly num_legacy_lines=$( (iptables-legacy-save || true; ip6tables-legacy-save || true) 2>/dev/null | grep -c '^-' || true)
readonly num_nft_lines=$( (timeout 5 sh -c "iptables-nft-save; ip6tables-nft-save" || true) 2>/dev/null | grep -c '^-' || true)


if [ "$num_legacy_lines" -gt 0 ] && [ "$num_nft_lines" -gt 0 ]; then
  echo "Found rules from both versions, iptables-legacy: ${num_legacy_lines} iptables-nft: ${num_nft_lines}"
  echo $NONOK
elif [ "$num_legacy_lines" -gt 0 ] && [ "$num_nft_lines" -eq 0 ]; then
  echo "Using iptables-legacy: ${num_legacy_lines} rules"
  echo $OK
elif [ "$num_legacy_lines" -eq 0 ] && [ "$num_nft_lines" -gt 0 ]; then
  echo "Using iptables-nft: ${num_nft_lines} rules"
  echo $OK
else
  echo "No iptables rules found"
  echo $UNKNOWN
fi
