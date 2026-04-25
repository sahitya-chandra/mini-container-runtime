package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func setupMemoryCgroup(pid int, rawLimit string) error {
	root := os.Getenv("CGROUP_ROOT")
	if root == "" {
		root = "/sys/fs/cgroup"
	}

	if _, err := os.Stat(filepath.Join(root, "cgroup.controllers")); err != nil {
		return fmt.Errorf("cgroup v2 not detected at %s (no cgroup.controllers): %w", root, err)
	}

	groupDir := filepath.Join(root, "mini-runc")
	if err := os.MkdirAll(groupDir, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup directory %s: %w", groupDir, err)
	}

	if _, err := strconv.ParseInt(rawLimit, 10, 64); err != nil {
		return fmt.Errorf("memory limit must be an integer number of bytes: %w", err)
	}

	if err := os.WriteFile(filepath.Join(groupDir, "memory.max"), []byte(rawLimit+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write memory.max: %w", err)
	}

	if err := os.WriteFile(filepath.Join(groupDir, "cgroup.procs"), []byte(strconv.Itoa(pid)+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to add pid to cgroup.procs: %w", err)
	}

	return nil
}
