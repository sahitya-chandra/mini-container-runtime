package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func run(args []string) {
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	rootfsFlag := runCmd.String("rootfs", "", "path to the container root filesystem")
	hostnameFlag := runCmd.String("hostname", "", "container hostname")

	if err := runCmd.Parse(args); err != nil {
		panic(err)
	}

	remaining := runCmd.Args()
	if len(remaining) < 1 {
		fmt.Println("you must provide a command to run inside the container")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  mini-runc run --rootfs=/path/to/rootfs --hostname=demo /bin/sh")
		return
	}

	rootfs := *rootfsFlag
	if rootfs == "" {
		rootfs = os.Getenv("ROOTFS_PATH")
	}
	if rootfs == "" {
		fmt.Println("no rootfs specified. Use --rootfs flag or set ROOTFS_PATH env")
		return
	}

	hostname := *hostnameFlag
	if hostname == "" {
		hostname = os.Getenv("CONTAINER_HOSTNAME")
	}
	if hostname == "" {
		hostname = "container"
	}

	env := os.Environ()
	env = append(env, "ROOTFS_PATH="+rootfs)
	env = append(env, "CONTAINER_HOSTNAME="+hostname)

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, remaining...)...)
	cmd.Env = env

	cloneFlags := uintptr(syscall.CLONE_NEWPID |
		syscall.CLONE_NEWNS |
		syscall.CLONE_NEWUTS |
		syscall.CLONE_NEWIPC |
		syscall.CLONE_NEWNET)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: cloneFlags,
	}

	if os.Getuid() != 0 {
		cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWUSER
		cmd.SysProcAttr.UidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		}
		cmd.SysProcAttr.GidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		}
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		f, err := pty.Start(cmd)
		if err != nil {
			panic(fmt.Sprintf("failed to start with PTY: %v", err))
		}
		defer f.Close()

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				if err := pty.InheritSize(os.Stdin, f); err != nil {
					fmt.Printf("error resizing pty: %s\n", err)
				}
			}
		}()
		ch <- syscall.SIGWINCH
		defer func() { signal.Stop(ch); close(ch) }()

		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		go func() { _, _ = io.Copy(f, os.Stdin) }()
		_, _ = io.Copy(os.Stdout, f)

		_ = cmd.Wait()
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			panic(err)
		}

		_ = cmd.Wait()
	}
}

