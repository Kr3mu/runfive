//go:build windows

package launcher

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func newProcessGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func terminateProcessTree(pid int) error {
	if pid <= 0 {
		return nil
	}
	return taskkill(pid, false)
}

func killProcessTree(pid int) error {
	if pid <= 0 {
		return nil
	}
	return taskkill(pid, true)
}

func taskkill(pid int, force bool) error {
	args := []string{"/PID", strconv.Itoa(pid), "/T"}
	if force {
		args = append(args, "/F")
	}

	//nolint:gosec // arguments are constructed from an integer PID and static flags.
	cmd := exec.Command("taskkill", args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	return fmt.Errorf("taskkill %d: %w: %s", pid, err, strings.TrimSpace(string(output)))
}
