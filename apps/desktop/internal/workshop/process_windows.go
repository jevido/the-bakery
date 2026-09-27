//go:build windows

package workshop

import (
	"os/exec"
	"syscall"
)

// ownProcessGroup starts the command in a group of its own.
func ownProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// interrupt has no gentle form on Windows for a process without a console;
// the run is ended at once.
func interrupt(cmd *exec.Cmd) { kill(cmd) }

func kill(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
