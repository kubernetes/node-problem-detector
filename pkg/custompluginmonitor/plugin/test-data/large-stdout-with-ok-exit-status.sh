#!/usr/bin/env bash

# Emits 16384 bytes of stdout (larger than the old hardcoded 4 KiB capture
# buffer and larger than the test's max_output_length) with a success exit
# code. Used to verify that plugin output capture honors max_output_length
# rather than a fixed internal buffer size.
head -c 16384 /dev/zero | tr '\0' 'a'
exit 0
