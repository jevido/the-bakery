package main

import (
	"fmt"
	"os"
	"os/exec"
)

// reexecAsBakeryUser re-execs the current process as the bakery system user
// via runuser (no password prompt needed, since this only ever runs while
// already root) and exits with its exit code. Subcommands that manage
// per-user state (systemd --user units, rootless Podman/Caddy) must run as
// bakery specifically — this lets an operator invoke them as root without
// remembering to `runuser -u bakery` first, matching jevido/bakery-agent's
// "commands that must run as bakery auto-re-execute" convention.
func reexecAsBakeryUser(args []string) {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bakery: could not resolve own executable path: %v\n", err)
		os.Exit(1)
	}

	runuserArgs := append([]string{"-u", bakeryUsername, "--", self}, args...)
	cmd := exec.Command("runuser", runuserArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "bakery: failed to re-exec as %s: %v\n", bakeryUsername, err)
		os.Exit(1)
	}
	os.Exit(0)
}

// ensureUserSessionEnv sets XDG_RUNTIME_DIR/DBUS_SESSION_BUS_ADDRESS
// explicitly before any systemctl --user/rootless Podman call — a
// non-interactive runuser/SSH session doesn't reliably inherit the D-Bus
// session pam_systemd sets up for a real login, even with lingering
// enabled (first hit, and documented, in the old root-bootstrap script's
// phase-2 script).
//
// Always overwrites rather than only setting when empty: `runuser -u
// bakery -- <self>` (reexecAsBakeryUser) doesn't clear the caller's
// environment, so when this runs after re-execing from root, it's not
// that XDG_RUNTIME_DIR is unset — it's that it's already set, to *root's*
// runtime dir (/run/user/0), inherited from the parent process. A
// present-but-wrong value looked "already set" to a bare emptiness check
// and was left alone, so every `systemctl --user`/rootless Podman call
// downstream tried to reach root's session bus instead of bakery's and
// failed with "Operation not permitted". Recomputing unconditionally from
// the current (by now already re-exec'd) UID is correct either way —
// harmless when it's already right, since it produces the same value.
func ensureUserSessionEnv() {
	runtimeDir := fmt.Sprintf("/run/user/%d", os.Getuid())
	os.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	os.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+runtimeDir+"/bus")
}
