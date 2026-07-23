package main

import (
	"fmt"
	"os"
	"testing"
)

// TestEnsureUserSessionEnvOverwritesInheritedValue guards against a real
// regression: `runuser -u bakery -- <self>` (reexecAsBakeryUser) doesn't
// clear the caller's environment, so re-execing from root leaves
// XDG_RUNTIME_DIR set to *root's* runtime dir (/run/user/0), inherited from
// the parent process — a present-but-wrong value that a bare "already set,
// leave it alone" check couldn't tell apart from a correct one. This is
// exactly the bug that broke `bakery bootstrap`/`bakery join` on a real box
// (systemctl --user failing with "Operation not permitted", reaching
// root's session bus instead of bakery's) once re-exec was actually
// exercised for real — see reexec.go's comment for the full story.
func TestEnsureUserSessionEnvOverwritesInheritedValue(t *testing.T) {
	// Simulate exactly what re-execing from root leaves behind: a value for
	// a *different* uid than the process now actually runs as.
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/0")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/run/user/0/bus")

	ensureUserSessionEnv()

	wantDir := fmt.Sprintf("/run/user/%d", os.Getuid())
	if got := os.Getenv("XDG_RUNTIME_DIR"); got != wantDir {
		t.Errorf("XDG_RUNTIME_DIR = %q, want %q (inherited root value was not overwritten)", got, wantDir)
	}
	wantBus := "unix:path=" + wantDir + "/bus"
	if got := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); got != wantBus {
		t.Errorf("DBUS_SESSION_BUS_ADDRESS = %q, want %q", got, wantBus)
	}
}
