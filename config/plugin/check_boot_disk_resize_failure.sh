#!/bin/bash

# This plugin checks if the resize2f partition failed during the boot process.

readonly OK=0
readonly NONOK=1
readonly UNKNOWN=2
readonly DISKFRACTIONMINIMUM=0.9
readonly ROOTDEVICE="sda1"

if ! grep $ROOTDEVICE /proc/partitions > /dev/null; then
  echo "Error retrieving requested disk size"
fi

readonly requestedDiskSize="$(grep $ROOTDEVICE /proc/partitions | awk 'NR == 1 {printf $3}')"

if ! df -P "/dev/$ROOTDEVICE"  > /dev/null; then
  echo "Error retrieving actual disk size"
fi

readonly actualDiskSize="$(df -P "/dev/$ROOTDEVICE" | awk 'NR == 2 {printf $2}')"

readonly ratio=$(echo "$actualDiskSize/$requestedDiskSize" | bc -l)

# if the ratio of actualdiskSize to requestedDiskSize is less than 0.9, then it
# implies there is a problem occuring during the resize2f partition.
if (( $(echo "$ratio < $DISKFRACTIONMINIMUM" | bc -l) )); then
    echo "DiskSizeCheck failure occured"
    exit $NONOK
else
    echo "DiskSizeCheck is successful"
    exit $OK
fi