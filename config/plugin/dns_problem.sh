#!/bin/bash

# This plugin checks for dns network issues.

readonly OK=0
readonly NONOK=1
readonly UNKNOWN=2

readonly KUBERNETES_SERVICE='kubernetes.default'

# Check getent command is present
if ! command -v getent >/dev/null; then
  echo "Could not find 'getent' - require getent"
  exit $UNKNOWN
fi

# Return success if a DNS lookup of the kubernetes service is successful
if getent hosts "${KUBERNETES_SERVICE}" >/dev/null; then
  echo "DNS lookup to ${KUBERNETES_SERVICE} is working"
  exit $OK
else
  echo "DNS lookup to ${KUBERNETES_SERVICE} is not working"
  exit $NONOK
fi
