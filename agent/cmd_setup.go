package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// requiredPackages mirrors the old root-bootstrap script's root-level package
// list: rootless Podman plus its helper binaries, ufw, and the utilities
// later setup/bootstrap steps shell out to.
var requiredPackages = []string{
	"podman",
	"uidmap",
	"slirp4netns",
	"fuse-overlayfs",
	"passt",
	"aardvark-dns",
	"ufw",
	"curl",
	"ca-certificates",
	"openssl",
	// Bun's official installer (ensureBunInstalled in build_worker.go,
	// Phase 08 task 14) needs unzip to extract its release archive — not
	// present on a minimal Debian 13 image. Found live: `bakery bootstrap`
	// failed provisioning the build worker without it.
	"unzip",
	// better-sqlite3 (an optional peer dep pulled in transitively by
	// better-auth/drizzle-orm, even though this app only ever uses
	// Postgres) has a native addon that needs compiling on `bun install`
	// for the build worker's checkout — same reasoning the Containerfile's
	// build stage already installs these for its own `bun install`. Found
	// live: `make`/`g++` weren't present on a minimal Debian 13 image
	// (python3 happened to already be there, but not documented/guaranteed,
	// so listed explicitly here too).
	"python3",
	"make",
	"g++",
}

const (
	unprivilegedPortsSysctlFile = "/etc/sysctl.d/50-bakery-unprivileged-ports.conf"
	unprivilegedPortsSysctlLine = "net.ipv4.ip_unprivileged_port_start=0"
	subIDRange                  = "100000-165535"
)

// cmdSetup idempotently installs everything a fresh box needs before any
// Podman/Quadlet work can happen: rootless Podman and its helper packages,
// subuid/subgid ranges for the bakery user, the unprivileged-port sysctl,
// and firewall rules. Ported from the old root-bootstrap script's root-level
// section, but safe to re-run (that script explicitly wasn't).
func cmdSetup(args []string) {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "bakery: setup must run as root")
		os.Exit(1)
	}

	if err := checkDebian13(); err != nil {
		fmt.Fprintf(os.Stderr, "bakery: setup: %v\n", err)
		os.Exit(1)
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{"installing OS packages", installPackages},
		{"configuring subuid/subgid ranges for the bakery user", configureSubIDRanges},
		{"allowing rootless binding to ports 80/443", allowUnprivilegedPorts},
		{"configuring firewall", configureFirewall},
	}

	for _, step := range steps {
		fmt.Printf("--- %s ---\n", step.name)
		if err := step.fn(); err != nil {
			fmt.Fprintf(os.Stderr, "bakery: setup: %s: %v\n", step.name, err)
			os.Exit(1)
		}
	}

	fmt.Println()
	fmt.Println("Setup complete: rootless Podman prerequisites, subuid/subgid ranges,")
	fmt.Println("unprivileged-port binding, and firewall rules are all in place.")
}

// checkDebian13 ports the old root-bootstrap script's OS check — requiredPackages
// below assumes apt-get/dpkg specifically, so a clear error here beats an
// oblique "apt-get: command not found" partway through installPackages.
func checkDebian13() error {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("reading /etc/os-release: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == `VERSION_ID="13"` {
			return nil
		}
	}
	return fmt.Errorf("this only supports Debian 13 (found: %s)", firstLineWithPrefix(string(data), "PRETTY_NAME="))
}

func firstLineWithPrefix(data, prefix string) string {
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return "unknown"
}

func installPackages() error {
	var missing []string
	for _, pkg := range requiredPackages {
		if !packageInstalled(pkg) {
			missing = append(missing, pkg)
		}
	}
	if len(missing) == 0 {
		fmt.Println("all required packages already installed")
		return nil
	}

	fmt.Printf("installing: %s\n", strings.Join(missing, " "))
	if err := runCmd("apt-get", "update"); err != nil {
		return fmt.Errorf("apt-get update: %w", err)
	}
	if err := runCmd("apt-get", append([]string{"install", "-y"}, missing...)...); err != nil {
		return fmt.Errorf("apt-get install: %w", err)
	}
	return nil
}

func packageInstalled(pkg string) bool {
	return exec.Command("dpkg", "-s", pkg).Run() == nil
}

// configureSubIDRanges ensures the bakery user has a subuid/subgid range,
// required for rootless Podman. A normal `useradd` on Debian 13
// auto-assigns one, but the install shim creates the bakery user with
// `--system` (task 04), which does not — so this can't be assumed and has
// to be checked/fixed explicitly.
func configureSubIDRanges() error {
	subuidOK, err := hasSubIDEntry("/etc/subuid")
	if err != nil {
		return err
	}
	subgidOK, err := hasSubIDEntry("/etc/subgid")
	if err != nil {
		return err
	}

	if subuidOK && subgidOK {
		fmt.Println("subuid/subgid ranges already configured")
		return nil
	}

	if err := runCmd("usermod",
		"--add-subuids", subIDRange,
		"--add-subgids", subIDRange,
		bakeryUsername,
	); err != nil {
		return fmt.Errorf("usermod --add-subuids/--add-subgids: %w", err)
	}
	return nil
}

func hasSubIDEntry(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, bakeryUsername+":") {
			return true, nil
		}
	}
	return false, nil
}

// allowUnprivilegedPorts lets rootless Podman/Caddy bind ports 80/443 —
// same fix src/lib/server/agent/install.sh applies best-effort for
// per-host Caddy; done proactively here since setup already has root.
func allowUnprivilegedPorts() error {
	current, err := os.ReadFile(unprivilegedPortsSysctlFile)
	if err == nil && strings.TrimSpace(string(current)) == unprivilegedPortsSysctlLine {
		fmt.Println("unprivileged-port sysctl already configured")
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", unprivilegedPortsSysctlFile, err)
	}

	if err := os.WriteFile(unprivilegedPortsSysctlFile, []byte(unprivilegedPortsSysctlLine+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", unprivilegedPortsSysctlFile, err)
	}
	if err := runCmd("sysctl", "--system"); err != nil {
		return fmt.Errorf("sysctl --system: %w", err)
	}
	return nil
}

// configureFirewall opens only 22/80/443, default-deny otherwise — the
// actual boundary keeping Postgres and the registry off the public
// internet, regardless of which interface(s) they publish to on the box
// itself (see postgres.go's writePostgresUnit for why Postgres can't
// stay strictly loopback-only despite that once being the intent here).
func configureFirewall() error {
	for _, port := range []string{"22/tcp", "80/tcp", "443/tcp"} {
		if ufwRuleExists(port) {
			continue
		}
		if err := runCmd("ufw", "allow", port); err != nil {
			return fmt.Errorf("ufw allow %s: %w", port, err)
		}
	}

	if !ufwActive() {
		if err := runCmd("ufw", "--force", "enable"); err != nil {
			return fmt.Errorf("ufw enable: %w", err)
		}
	}
	return nil
}

func ufwRuleExists(port string) bool {
	out, err := exec.Command("ufw", "status").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), port)
}

func ufwActive() bool {
	out, err := exec.Command("ufw", "status").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Status: active")
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
