#!/bin/sh

# ReadonlyFilesystem detector with auto-recovery capability.
#
# Strategy:
# 1. Find all devices remounted read-only in last 5 minutes (from /dev/kmsg)
# 2. Check if ANY of those devices are CURRENTLY mounted read-only
# 3. If ANY device is RO -> set ReadonlyFilesystem=True (exit 1)
# 4. Get all devices remounted read-only in history (from /dev/kmsg)
# 5. Check if ALL of those devices are CURRENTLY mounted read-only OR not mounted at all
# 6. If ALL devices recovered -> set ReadonlyFilesystem=False (exit 0)


readonly OK=0 # All devices recovered, clear condition (condition: False)
readonly NONOK=1 # At least one device still RO (condition: True)
readonly MOUNTS_FILE_HOST="/host/proc/1/mounts"
readonly MOUNTS_FILE_LOCAL="/proc/mounts"
readonly LOOKBACK_SEC=300  # 5 minutes lookback

# Extract device name from kernel message
# Example: "EXT4-fs (sda1): remounting filesystem read-only"
# Example: "XFS (dm-3): remounting filesystem read-only"
extract_device_name() {
    _msg="$1"
    # Extract device from parentheses: (device)
    _device=$(printf '%s\n' "$_msg" | sed -n 's/.*(\([^)]*\)).*/\1/p')
    if [ -n "$_device" ]; then
        printf '%s\n' "$_device"
        return 0
    fi
    return 1
}

# Extract device names from kmsg output
# Input: kmsg messages (one per line)
# Output: unique device names (sorted)
extract_devices_from_messages() {
    _messages="$1"
    printf '%s\n' "$_messages" | while IFS= read -r _line; do
        # Skip empty lines
        [ -z "$_line" ] && continue

        # Extract device name from the message
        _dev=$(extract_device_name "$_line")
        if [ -n "$_dev" ]; then
            printf '%s\n' "$_dev"
        fi
    done | sort -u
}

# Get all possible device paths for a given device name
# Handles dm-X devices by resolving symlinks via udevadm
# Returns: space-separated list of device paths to check
get_device_paths() {
    _dev="$1"
    _paths="$_dev"

    # Add common path prefixes
    _paths="$_paths /dev/$_dev"

    # For dm-X devices, get symlinks from udevadm
    case "$_dev" in
        dm-*)
            # Try to get symlinks for this dm device
            if command -v udevadm >/dev/null 2>&1; then
                _symlinks=$(udevadm info --query=symlink --name="/dev/$_dev" 2>/dev/null || true)
                if [ -n "$_symlinks" ]; then
                    # Add each symlink as a potential path
                    for _link in $_symlinks; do
                        _paths="$_paths /dev/$_link"
                    done
                fi
            fi
            ;;
    esac

    # For devices with ! character (Portworx dm-name format)
    # Example: pxd!123 → /dev/mapper/pxd!123
    # Example: pxd!pxd952712810427059188 → /dev/mapper/pxd!pxd952712810427059188 AND /dev/pxd/pxd952712810427059188
    case "$_dev" in
        *!*)
            _paths="$_paths /dev/mapper/$_dev"

            # If pattern is pxd!pxdXXX, also check /dev/pxd/pxdXXX
            # Extract the part after ! and check if it starts with pxd followed by numbers
            _after_bang="${_dev#*!}"
            case "$_after_bang" in
                pxd[0-9]*)
                    # Add /dev/pxd/pxdXXX path
                    _paths="$_paths /dev/pxd/$_after_bang"
                    ;;
            esac
            ;;
    esac

    # For devices with - character that might be in /dev/mapper/
    # Example: pwx0-206233844786798552 → /dev/mapper/pwx0-206233844786798552
    # Example: 3624a93704cdc47b41e974dd913a8eac2 → /dev/mapper/3624a93704cdc47b41e974dd913a8eac2
    case "$_dev" in
        *-*|[0-9a-f][0-9a-f][0-9a-f][0-9a-f]*)
            # Device name contains - or looks like a WWID (hex string)
            # These are often in /dev/mapper/
            _paths="$_paths /dev/mapper/$_dev"
            ;;
    esac

    # For Portworx pxd devices in /dev/pxd/ directory
    # Example: pxd342462708072724230 → /dev/pxd/pxd342462708072724230
    case "$_dev" in
        pxd[0-9]*)
            # Device name starts with pxd followed by numbers
            _paths="$_paths /dev/pxd/$_dev"
            ;;
    esac

    printf '%s\n' "$_paths"
}

# Check if device is currently mounted read-only
# Returns: 0 if device is RO, 1 if device is RW or not mounted
is_device_readonly() {
    _device="$1"
    _mounts_file="$MOUNTS_FILE_HOST"

    # Try host mounts first, fallback to local
    [ ! -r "$_mounts_file" ] && _mounts_file="$MOUNTS_FILE_LOCAL"

    if [ ! -r "$_mounts_file" ]; then
        return 1  # Cannot determine, assume not RO
    fi

    # Get all possible device paths for this device
    _device_paths=$(get_device_paths "$_device")

    # Parse /proc/mounts to check current state
    # Format: device mountpoint fstype options dump pass
    while IFS=' ' read -r _mount_device _mountpoint _fstype _options _rest; do
        _match=0

        # Check if mount device matches any of our device paths
        for _path in $_device_paths; do
            if [ "$_mount_device" = "$_path" ]; then
                _match=1
                break
            fi
        done

        # If device matched, check if it's mounted read-only
        if [ $_match -eq 1 ]; then
            case ",$_options," in
                *,ro,*)
                    printf 'Device %s at %s is read-only\n' "$_mount_device" "$_mountpoint"
                    return 0  # Device is RO
                    ;;
            esac
        fi
    done < "$_mounts_file"

    # Device not found or is RW
    return 1
}
printf 'Scanning /dev/kmsg for '\''Remounting filesystem read-only'\'' messages...\n'


# Step 1: Get devices from /dev/kmsg with 5-minute lookback for DETECTION
# Check if /dev/kmsg is readable
if [ ! -r /dev/kmsg ]; then
    printf 'Warning: /dev/kmsg not readable, ReadonlyFilesystem condition: False\n'
    exit $OK
fi

# Calculate cutoff timestamp in microseconds for 5-minute lookback
# /proc/uptime -> seconds since boot (float). Convert to microseconds and subtract lookback.
if [ -f /proc/uptime ] && [ -r /proc/uptime ]; then
    CUTOFF_US=$(awk -v lb="$LOOKBACK_SEC" '
      NR==1 {
        up = $1 + 0
        c = (up - lb) * 1000000
        if (c < 0) c = 0
        printf("%.0f\n", c)
        exit
      }' /proc/uptime 2>/dev/null)

    # Check if awk succeeded
    if [ -z "$CUTOFF_US" ]; then
        CUTOFF_US=0
    fi
else
    # Fallback: if /proc/uptime not available, set cutoff to 0 (get all messages)
    CUTOFF_US=0
fi

# Filter /dev/kmsg by timestamp (only messages within 5-minute lookback period for DETECTION)
# Use timeout with cat to read all available messages then stop
if command -v timeout >/dev/null 2>&1; then
    kmsg_output_recent=$(timeout 10 cat /dev/kmsg 2>/dev/null | awk -v cutoff="$CUTOFF_US" '
      BEGIN { keep=0 }
      /^[ \t]/ { if (keep) print; next }
      {
        semi = index($0, ";"); if (!semi) next
        header = substr($0, 1, semi-1)
        msg    = substr($0, semi+1)
        n = split(header, h, ","); if (n < 3) { keep=0; next }
        ts = h[3] + 0
        keep = (ts >= cutoff)
        if (keep) print msg
      }' 2>/dev/null | grep -iE "remounting filesystem read-only" 2>/dev/null || true)
else
    # Fallback: if no timeout command, skip detection
    printf 'Warning: timeout command not available, cannot safely read /dev/kmsg\n'
    kmsg_output_recent=""
fi

# Extract device names from recent messages (5-minute lookback)
devices_recent=$(extract_devices_from_messages "$kmsg_output_recent")

# Step 2: Check current mount state
# If recent devices (5-min) found → check them and set to True if RO
# If no recent devices but old devices exist → check old devices for recovery

if [ -n "$devices_recent" ]; then
    any_device_ro=0
    for _dev in $devices_recent; do
        if is_device_readonly "$_dev"; then
            any_device_ro=1
        fi
    done

    printf '\n'
    if [ $any_device_ro -eq 1 ]; then
        printf 'At least one device is currently read-only. ReadonlyFilesystem condition: True\n'
        exit $NONOK  # Exit 1 = Condition True
    fi
fi

# Step 3: Get ALL devices ever mentioned in /dev/kmsg (no time limit) for RECOVERY check
# Scanning /dev/kmsg for ALL 'Remounting filesystem read-only' messages (no time limit for recovery check)

# Get ALL messages from /dev/kmsg (no timestamp filtering)
# Use timeout with cat to read all available messages then stop
if command -v timeout >/dev/null 2>&1; then
    kmsg_output_all=$(timeout 10 cat /dev/kmsg 2>/dev/null | awk '
      BEGIN { keep=0 }
      /^[ \t]/ { if (keep) print; next }
      {
        semi = index($0, ";"); if (!semi) next
        msg = substr($0, semi+1)
        print msg
      }' 2>/dev/null | grep -iE "remounting filesystem read-only" 2>/dev/null || true)
else
    # Fallback: if no timeout command, skip recovery check
    kmsg_output_all=""
fi

# Extract ALL device names ever mentioned
devices_all=$(extract_devices_from_messages "$kmsg_output_all")

# Step 4: Check if any old devices are still RO
if [ -n "$devices_all" ]; then
    any_device_ro=0
    for _dev in $devices_all; do
        if is_device_readonly "$_dev"; then
            any_device_ro=1
        fi
    done

    printf '\n'
    if [ $any_device_ro -eq 1 ]; then
        printf 'At least one device is still read-only, ReadonlyFilesystem condition: True\n'
        exit $NONOK  # Exit 1 = Condition True (not recovered)
    else
        printf 'All devices have recovered, ReadonlyFilesystem condition: False\n'
        exit $OK  # Exit 0 = Condition False (recovered!)
    fi
else
    printf 'No '\''Remounting filesystem read-only'\'' messages found in kmsg, ReadonlyFilesystem condition: False\n'
    exit $OK
fi
