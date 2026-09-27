package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// replaceAppImage moves the verified new AppImage over the running one and
// starts it. The running app keeps its old copy mounted until it quits.
func (s *UpdateService) replaceAppImage(target, staged string) error {
	tmp := target + ".update"
	if err := copyFile(staged, tmp, 0o755); err != nil {
		return fmt.Errorf("writing the update next to %s: %w", filepath.Base(target), err)
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replacing %s: %w", filepath.Base(target), err)
	}
	// Descriptors GTK, WebKit or the AppImage runtime opened without
	// close-on-exec would leak into the new version and keep this AppImage's
	// mount alive after we quit. We quit right after, so close them all.
	markAllCloseOnExec()
	cmd := exec.Command(target)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Env = withoutAppImageEnv(os.Environ())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting the new version: %w", err)
	}
	s.app.Quit()
	return nil
}

func markAllCloseOnExec() {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return
	}
	for _, e := range entries {
		if fd, err := strconv.Atoi(e.Name()); err == nil && fd > 2 {
			syscall.CloseOnExec(fd)
		}
	}
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// withoutAppImageEnv drops the variables the AppImage runtime set for this
// instance, so the new one sets its own.
func withoutAppImageEnv(env []string) []string {
	out := env[:0:0]
	for _, kv := range env {
		switch {
		case hasPrefix(kv, "APPIMAGE="), hasPrefix(kv, "APPDIR="), hasPrefix(kv, "OWD="), hasPrefix(kv, "ARGV0="):
			continue
		}
		out = append(out, kv)
	}
	return out
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
