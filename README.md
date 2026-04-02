# mini-runc (learning project)

Tiny container runtime in Go for my own learning.  
Focus: namespaces, `pivot_root`, `/dev`/TTYs, and a bit of cgroups. 

## What this tool does

- Creates a new process in **separate Linux namespaces**:
  - PID, mount, UTS (hostname), IPC, and optionally user.
- Switches to a **new root filesystem** (`rootfs`) using `pivot_root`.
- Mounts basic pseudo-filesystems: `/proc`, `/sys`, and `/dev`.
- Sets up a **PTY** so that shells inside the container behave like you expect.
- Forwards signals (Ctrl+C, etc.) from the host terminal into the container process.

## How to build

```bash
go build -o mini-runc .
```

Linux-only; user namespaces must be enabled for unprivileged use.

## Preparing a root filesystem (rootfs)

The program expects a **root filesystem directory** containing at least:

- Basic directories: `/bin`, `/lib`, `/lib64`, `/usr`, `/etc`, `/dev`, `/proc`, `/sys`, `/tmp`, …
- A shell like `/bin/sh` inside that rootfs.

For learning, you can:

- Use a minimal **busybox** or **alpine** rootfs (e.g. extract an alpine tarball into `rootfs/`).
- Or copy some parts of your host system into a directory (less clean, but OK for experiments).

Example (pseudo-steps, not exact commands):

```bash
mkdir -p rootfs
# e.g. download a minimal rootfs tarball and extract it:
#   tar -xpf alpine-minirootfs.tar.gz -C rootfs
```

Then point mini-runc at this directory.

## Running a container

You can configure the rootfs and hostname via:

- Flags: `--rootfs`, `--hostname`
- Or environment variables: `ROOTFS_PATH`, `CONTAINER_HOSTNAME`

Examples:

```bash
# Using environment variable for rootfs
export ROOTFS_PATH=/home/you/mini-runc/rootfs
go build -o mini-runc .
./mini-runc run /bin/sh

# Using explicit flags
./mini-runc run \
  --rootfs=/home/you/mini-runc/rootfs \
  --hostname=mybox \
  /bin/sh
```

Once inside, try:

```bash
hostname
ps aux
mount
tty
```

and compare the output with what you see on the host.

## CLI overview

```text
mini-runc - a tiny container runtime for learning

Usage:
  mini-runc run [flags] <command> [args...]

Subcommands:
  run     create a new container and run a command inside it

Common flags for 'run':
  --rootfs   path to the container root filesystem (default from ROOTFS_PATH env)
  --hostname container hostname (default: container or CONTAINER_HOSTNAME env)
```

## Namespaces used (and why)

mini-runc uses several Linux namespaces together:

- **PID namespace (`CLONE_NEWPID`)**  
  The container sees its own PID 1 and process tree. This lets you kill / inspect processes inside without touching the host's tree.

- **Mount namespace (`CLONE_NEWNS`)**  
  Mounts and unmounts you do inside do not affect the host. This is what allows us to `pivot_root` and mount `/proc`, `/sys`, `/dev` only inside the container.

- **UTS namespace (`CLONE_NEWUTS`)**  
  Gives the container its own hostname. You can run `hostname` inside and get a different name than on the host.

- **IPC namespace (`CLONE_NEWIPC`)**  
  Separates System V IPC (shared memory, semaphores, message queues), so IPC objects from the container are not visible globally.

- **Network namespace (`CLONE_NEWNET`)**  
  Gives the container its own network stack (interfaces, routes, firewall rules). mini-runc only brings up the loopback interface and tries to assign `127.0.0.2/8` on `lo` so you can see that the container has its own network namespace. For more advanced networking (veth pairs, bridges, NAT), you would typically configure this from the host.

- **User namespace (`CLONE_NEWUSER`, when not root)**  
  Maps UID 0 inside the container to your real unprivileged UID on the host. This means you feel like root inside, but kernel permission checks still treat you as your real user.

## Key concepts (very short)

- **Namespaces**: kernel feature that gives a process its **own view** of things like PIDs, mounts, hostname, IPC. This lets the container see a different world than the host.
- **User namespaces**: map container root (UID 0) to your real UID so you feel like root inside but remain unprivileged outside.
- **`pivot_root` vs `chroot`**: `pivot_root` fully swaps the root filesystem and hides the old root under `/oldroot`, which we then unmount. This is closer to what real containers do.
- **Pseudo-filesystems**: `/proc`, `/sys`, `/dev`, and `devpts` are special filesystems the kernel provides; we mount them inside the container so tools like `ps`, `top`, shells, and TTYs work.
- **PTY + TTY handling**: a pseudo-terminal connects your host terminal to the shell inside the container so interactive programs behave correctly.

## Minimal experiments / tests

These are simple manual tests you can run to verify isolation.

- **Hostname is different inside vs outside**

  ```bash
  # On the host
  hostname

  # Inside the container
  ./mini-runc run --rootfs=/path/to/rootfs --hostname=demo /bin/sh -lc 'hostname'
  ```

- **PID 1 is different inside vs outside**

  ```bash
  # On the host
  echo "host PID 1:"; cat /proc/1/status | grep ^Name

  # Inside the container
  ./mini-runc run --rootfs=/path/to/rootfs /bin/sh -lc 'echo "container PID 1:"; cat /proc/1/status | grep ^Name'
  ```

- **PID namespace is different (`/proc/1/ns/pid`)**

  ```bash
  # On the host
  readlink /proc/1/ns/pid

  # Inside the container
  ./mini-runc run --rootfs=/path/to/rootfs /bin/sh -lc 'readlink /proc/1/ns/pid'
  ```

- **Network namespace experiment (loopback)**

  Inside the container, assuming `ip` exists in the rootfs:

  ```bash
  ./mini-runc run --rootfs=/path/to/rootfs /bin/sh -lc 'ip addr show lo'
  ```

  You should see `127.0.0.1/8` and (if the `ip` commands in `child()` succeeded) `127.0.0.2/8` on `lo`, independent from any host configuration.
