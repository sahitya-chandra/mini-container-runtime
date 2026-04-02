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

echo "== Cgroup v2 memory limit demo =="
echo "Attempting to run a memory-limited container (128MiB)..."
"$BIN" run --rootfs="$ROOTFS" --memory=134217728 /bin/sh -lc '
  echo "Inside container: memory.max:"
  if [ -f /sys/fs/cgroup/memory.max ]; then
    cat /sys/fs/cgroup/memory.max
  else
    echo "memory.max not found (cgroup v2 may not be mounted here)"
  fi

  echo
  echo "Trying to allocate memory with a simple stress test (if available)..."
  if command -v stress >/dev/null 2>&1; then
    echo "Running: stress --vm 1 --vm-bytes 256M --vm-hang 0 --timeout 5"
    stress --vm 1 --vm-bytes 256M --vm-hang 0 --timeout 5 || echo "stress failed or was killed (possibly by cgroup limit)"
  else
    echo "stress binary not found in rootfs; skipping memory pressure demo."
  fi
'
echo

echo "All tests executed. Inspect the output above to understand the isolation and resource limits."

