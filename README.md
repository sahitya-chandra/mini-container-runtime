# mini-runc

Tiny container runtime in Go. Built to learn how namespaces, `pivot_root`, `/dev`, TTYs, and networking work under the hood.

## Build & run

```bash
go build -o mini-runc .

# with flags
./mini-runc run --rootfs=/path/to/rootfs --hostname=demo /bin/sh

# or with env
export ROOTFS_PATH=/path/to/rootfs
./mini-runc run /bin/sh
```

Linux-only. User namespaces must be enabled for unprivileged use.

## Rootfs setup

Extract a minimal alpine rootfs (or similar) into a directory:

```bash
mkdir -p rootfs
tar -xpf alpine-minirootfs-*.tar.gz -C rootfs
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--rootfs` | `ROOTFS_PATH` env | Path to the container root filesystem |
| `--hostname` | `container` | Hostname inside the container |

## What it uses

| Namespace | Flag | Purpose |
|-----------|------|---------|
| PID | `CLONE_NEWPID` | Isolated process tree |
| Mount | `CLONE_NEWNS` | Independent mounts, `pivot_root` |
| UTS | `CLONE_NEWUTS` | Separate hostname |
| IPC | `CLONE_NEWIPC` | Isolated SysV IPC |
| Network | `CLONE_NEWNET` | Own loopback, separate net stack |
| User | `CLONE_NEWUSER` | UID 0 inside maps to your real UID (when not root) |

## Quick verification

```bash
# hostname isolation
./mini-runc run --rootfs=rootfs --hostname=box /bin/sh -lc 'hostname'

# PID namespace
./mini-runc run --rootfs=rootfs /bin/sh -lc 'echo PID ns: $(readlink /proc/1/ns/pid)'

# network namespace
./mini-runc run --rootfs=rootfs /bin/sh -lc 'ip addr show lo'
```

## Project structure

```
main.go        - CLI entry point
run.go         - parent process: flag parsing, namespace setup, PTY
container.go   - child process: pivot_root, mounts, /dev, signal forwarding
tests.sh       - manual integration tests
```
