//go:build windows
// +build windows

package runner

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr configures the subprocess on Windows.
// Windows doesn't have Setpgid; we use CREATE_NEW_PROCESS_GROUP instead.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
