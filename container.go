package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func child() {
	args := os.Args[2:]

	hostPtyPath, _ := os.Readlink("/proc/self/fd/0")

	must(syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""))

	rootfs := os.Getenv("ROOTFS_PATH")
	if rootfs == "" {
		panic("ROOTFS_PATH not set")
	}
	absRootfs, err := filepath.Abs(rootfs)
	must(err)
	rootfs = absRootfs

	must(syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""))

	oldroot := filepath.Join(rootfs, "oldroot")
	must(os.MkdirAll(oldroot, 0700))

	must(syscall.PivotRoot(rootfs, oldroot))
	must(os.Chdir("/"))

	if err := exec.Command("ip", "link", "set", "lo", "up").Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to bring up lo: %v\n", err)
	}
	if err := exec.Command("ip", "addr", "add", "127.0.0.2/8", "dev", "lo").Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add 127.0.0.2: %v\n", err)
	}

	must(syscall.Mount("proc", "/proc", "proc", 0, ""))
	must(syscall.Mount("sysfs", "/sys", "sysfs", syscall.MS_RDONLY, ""))
	must(syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_NOEXEC, "mode=755"))

	if hostPtyPath != "" && strings.HasPrefix(hostPtyPath, "/dev/pts/") {
		f, _ := os.Create("/dev/console")
		if f != nil {
			f.Close()
			fullHostPtyPath := filepath.Join("/oldroot", hostPtyPath)
			if err := syscall.Mount(fullHostPtyPath, "/dev/console", "", syscall.MS_BIND, ""); err != nil {
				must(syscall.Mknod("/dev/console", syscall.S_IFCHR|0600, int(unix.Mkdev(5, 1))))
			}
		}
	} else {
		must(syscall.Mknod("/dev/console", syscall.S_IFCHR|0600, int(unix.Mkdev(5, 1))))
	}

	must(syscall.Mknod("/dev/null", syscall.S_IFCHR|0666, int(unix.Mkdev(1, 3))))
	must(syscall.Mknod("/dev/zero", syscall.S_IFCHR|0666, int(unix.Mkdev(1, 5))))
	must(syscall.Mknod("/dev/full", syscall.S_IFCHR|0666, int(unix.Mkdev(1, 7))))
	must(syscall.Mknod("/dev/random", syscall.S_IFCHR|0666, int(unix.Mkdev(1, 8))))
	must(syscall.Mknod("/dev/urandom", syscall.S_IFCHR|0666, int(unix.Mkdev(1, 9))))

	must(os.Symlink("/dev/console", "/dev/tty"))

	must(os.MkdirAll("/dev/pts", 0755))
	must(syscall.Mount("devpts", "/dev/pts", "devpts", 0, "newinstance,ptmxmode=0666,mode=0620"))
	must(os.Symlink("/dev/pts/ptmx", "/dev/ptmx"))

	must(os.Symlink("/proc/self/fd", "/dev/fd"))
	must(os.Symlink("/proc/self/fd/0", "/dev/stdin"))
	must(os.Symlink("/proc/self/fd/1", "/dev/stdout"))
	must(os.Symlink("/proc/self/fd/2", "/dev/stderr"))

	var origPgrp int
	isTTY := term.IsTerminal(0)
	if isTTY {
		if pgrp, err := unix.IoctlGetInt(0, unix.TIOCGPGRP); err == nil {
			origPgrp = pgrp
		}
		if err := unix.IoctlSetInt(0, unix.TIOCSCTTY, 1); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set TIOCSCTTY: %v\n", err)
		}
	}

	must(syscall.Unmount("/oldroot", syscall.MNT_DETACH))
	must(os.RemoveAll("/oldroot"))

	hostname := os.Getenv("CONTAINER_HOSTNAME")
	if hostname == "" {
		hostname = "container"
	}
	must(syscall.Sethostname([]byte(hostname)))

	cmdPath, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Printf("Command not found: %s\n", args[0])
		os.Exit(1)
	}

	pid, err := syscall.ForkExec(
		cmdPath,
		args,
		&syscall.ProcAttr{
			Env:   os.Environ(),
			Files: []uintptr{0, 1, 2},
		},
	)
	must(err)

	syscall.Setpgid(pid, pid)
	if isTTY {
		_ = unix.IoctlSetInt(0, unix.TIOCSPGRP, pid)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	go func() {
		for sig := range sigCh {
			_ = syscall.Kill(pid, sig.(syscall.Signal))
		}
	}()

	var exitCode int
	for {
		var ws syscall.WaitStatus
		wpid, err := syscall.Wait4(-1, &ws, 0, nil)
		if err != nil {
			if err == syscall.ECHILD {
				break
			}
			continue
		}
		if wpid == pid {
			if ws.Exited() {
				exitCode = ws.ExitStatus()
			} else if ws.Signaled() {
				exitCode = 128 + int(ws.Signal())
			}
		}
	}

	if isTTY && origPgrp != 0 {
		_ = unix.IoctlSetInt(0, unix.TIOCSPGRP, origPgrp)
	}

	os.Exit(exitCode)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
