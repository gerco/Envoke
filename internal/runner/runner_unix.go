//go:build !windows
// +build !windows

package runner

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr configures the subprocess to run in a new process group on Unix systems.
// This ensures signals (Ctrl-C, Ctrl-Z) go to the child, not envoke.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
