#!/usr/bin/env bash
set -euo pipefail

# Simple manual test runner for mini-runc.
# Adjust ROOTFS to point at your container root filesystem before running.

ROOTFS="${ROOTFS:-/path/to/rootfs}"
BIN="${BIN:-./mini-runc}"

if [[ ! -x "$BIN" ]]; then
  echo "Binary '$BIN' not found or not executable. Build it first with:"
  echo "  go build -o mini-runc ."
  exit 1
fi

echo "Using rootfs: $ROOTFS"
echo "Using binary: $BIN"
echo

echo "== Hostname test =="
echo "Host hostname: $(hostname)"
echo "Container hostname:"
"$BIN" run --rootfs="$ROOTFS" --hostname=demo /bin/sh -lc 'hostname'
echo

echo "== PID 1 name comparison =="
echo "Host PID 1:"
grep ^Name /proc/1/status
echo "Container PID 1:"
"$BIN" run --rootfs="$ROOTFS" /bin/sh -lc 'grep ^Name /proc/1/status'
echo

echo "== PID namespace handle =="
echo "Host /proc/1/ns/pid:   $(readlink /proc/1/ns/pid)"
echo "Container /proc/1/ns/pid:"
"$BIN" run --rootfs="$ROOTFS" /bin/sh -lc 'readlink /proc/1/ns/pid'
echo

echo "== Network namespace (loopback) =="
echo "Container lo interface:"
"$BIN" run --rootfs="$ROOTFS" /bin/sh -lc 'ip addr show lo || echo "no ip binary in rootfs"'
echo

echo "All tests executed. Inspect the output above to understand the isolation differences."

