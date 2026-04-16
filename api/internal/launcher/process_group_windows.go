//go:build windows

package launcher

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// taskkillTimeout bounds how long we wait for the taskkill child before we
// consider the kill attempt itself stuck. The caller already wraps this in a
// larger Stop() budget, so this just prevents a runaway taskkill process.
const taskkillTimeout = 5 * time.Second

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

	ctx, cancel := context.WithTimeout(context.Background(), taskkillTimeout)
	defer cancel()

	//nolint:gosec // arguments are constructed from an integer PID and static flags.
	cmd := exec.CommandContext(ctx, "taskkill", args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	return fmt.Errorf("taskkill %d: %w: %s", pid, err, strings.TrimSpace(string(output)))
}
