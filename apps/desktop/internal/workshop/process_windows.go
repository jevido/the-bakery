//go:build windows

package workshop

import (
	"os"
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

// KillGroup ends a process left behind by an app that went away during a
// run.
func KillGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil && pid > 0 {
		_ = p.Kill()
	}
}

func kill(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
