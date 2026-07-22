package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// daemonBinaryPath is where the install shim (task 04) installs the
// bakery binary system-wide — fixed, since the bakery user is now a fixed
// system account rather than an arbitrary one, unlike today's
// src/lib/server/agent/install.sh's per-user $HOME/.local/bin/bakery-agent.
const daemonBinaryPath = "/usr/local/bin/bakery"

// cmdJoin ports src/lib/server/agent/install.sh's per-host enrollment flow
// into Go: writes the daemon's token/URL config, installs its systemd
// --user unit, and sets up the per-host Caddy Quadlet — everything that
// script does once the binary is already on disk.
func cmdJoin(args []string) {
	fs := flag.NewFlagSet("bakery join", flag.ExitOnError)
	token := fs.String("token", "", "per-host bearer token issued by the control plane (required)")
	url := fs.String("url", "", "control-plane base URL (required)")
	fs.Parse(args)

	if *token == "" || *url == "" {
		fmt.Fprintln(os.Stderr, "Usage: bakery join --token=<token> --url=<bakery-url>")
		os.Exit(2)
	}

	if os.Geteuid() == 0 {
		reexecAsBakeryUser(os.Args[1:])
		return
	}

	ensureUserSessionEnv()

	home, err := os.UserHomeDir()
	if err != nil {
		joinFail("resolving home directory", err)
	}

	if err := writeDaemonConfig(home, *token, *url); err != nil {
		joinFail("writing daemon config", err)
	}
	if err := installDaemonUnit(home); err != nil {
		joinFail("installing daemon systemd unit", err)
	}
	fmt.Println()
	fmt.Println("bakery daemon installed and started.")
	fmt.Println("  Status: systemctl --user status bakery-daemon")
	fmt.Println("  Logs:   journalctl --user -u bakery-daemon -f")

	if err := setupCaddyQuadlet(home); err != nil {
		fmt.Fprintf(os.Stderr, "\nWARNING: Caddy setup failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "This does not affect the bakery daemon, which is already running.")
	} else {
		fmt.Println()
		fmt.Println("bakery-managed Caddy reverse proxy installed and started.")
		fmt.Println("  Status: systemctl --user status caddy")
		fmt.Println("  Logs:   journalctl --user -u caddy -f")
	}

	bestEffortUnprivilegedPorts()
}

func joinFail(step string, err error) {
	fmt.Fprintf(os.Stderr, "bakery: join: %s: %v\n", step, err)
	os.Exit(1)
}

func writeDaemonConfig(home, token, url string) error {
	configDir := filepath.Join(home, ".config", "bakery")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	envPath := filepath.Join(configDir, "agent.env")
	content := fmt.Sprintf("BAKERY_TOKEN=%s\nBAKERY_URL=%s\n", token, url)
	return os.WriteFile(envPath, []byte(content), 0o600)
}

func installDaemonUnit(home string) error {
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return err
	}

	envPath := filepath.Join(home, ".config", "bakery", "agent.env")
	unitPath := filepath.Join(unitDir, "bakery-daemon.service")

	unit := fmt.Sprintf(`[Unit]
Description=Bakery Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=%s
ExecStart=%s daemon
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`, envPath, daemonBinaryPath)

	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return err
	}

	if err := runCmd("systemctl", "--user", "daemon-reload"); err != nil {
		return err
	}
	return runCmd("systemctl", "--user", "enable", "--now", "bakery-daemon")
}

// setupCaddyQuadlet writes the per-host Caddy Quadlet unit and seeds an
// empty managed Caddyfile if none exists yet. Re-running join (e.g. to
// reinstall the daemon) must never clobber site config later deploys have
// since written for live domains — same care today's install.sh takes.
func setupCaddyQuadlet(home string) error {
	quadletDir := filepath.Join(home, ".config", "containers", "systemd")
	caddyConfigDir := filepath.Join(home, ".config", "bakery", "caddy")
	caddyDataDir := filepath.Join(home, ".local", "share", "bakery", "caddy")
	caddyfilePath := filepath.Join(caddyConfigDir, "Caddyfile")
	caddyUnitPath := filepath.Join(quadletDir, "caddy.container")

	for _, dir := range []string{quadletDir, caddyConfigDir, caddyDataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(caddyfilePath); os.IsNotExist(err) {
		skeleton := "# Managed by bakery-agent. Site blocks for deployed apps and custom\n" +
			"# domains are added here automatically — avoid hand-editing.\n"
		if err := os.WriteFile(caddyfilePath, []byte(skeleton), 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Network=host (not PublishPort) — Caddy has to dial app units by the
	// same 127.0.0.1:<port> the agent's own health probe uses, and app
	// units publish those ports to the host's real network namespace, not
	// a per-container one. A separate netns can't reach those.
	unit := fmt.Sprintf(`[Unit]
Description=Bakery Reverse Proxy (Caddy)
After=network-online.target

[Container]
Image=docker.io/library/caddy:2
Network=host
Volume=%s:/etc/caddy:Z
Volume=%s:/data:Z
AutoUpdate=registry

[Service]
Restart=always
TimeoutStartSec=90

[Install]
WantedBy=default.target
`, caddyConfigDir, caddyDataDir)

	if err := os.WriteFile(caddyUnitPath, []byte(unit), 0o644); err != nil {
		return err
	}

	// Quadlet-generated units live in systemd's generator output dir, not a
	// persistent unit file location — `systemctl enable` refuses them
	// ("transient or generated"). The `[Install]` section above is honored
	// by the generator itself at daemon-reload (it already symlinks into
	// default.target.wants/), so `start` is the only step needed, matching
	// commands.go's daemon-reload+start pattern for app Quadlet units.
	if err := runCmd("systemctl", "--user", "daemon-reload"); err != nil {
		return err
	}
	return runCmd("systemctl", "--user", "start", "caddy.service")
}

// bestEffortUnprivilegedPorts is a fallback for hosts where `bakery setup`
// wasn't run first (setup already does this proactively with root). Same
// best-effort spirit as today's install.sh: a `curl | sh`-driven join may
// have no sudo access at all, and that must not fail the join that already
// succeeded above.
func bestEffortUnprivilegedPorts() {
	current := 1024
	if out, err := exec.Command("sysctl", "-n", "net.ipv4.ip_unprivileged_port_start").Output(); err == nil {
		if v, convErr := strconv.Atoi(strings.TrimSpace(string(out))); convErr == nil {
			current = v
		}
	}
	if current <= 80 {
		return
	}

	if exec.Command("sudo", "-n", "true").Run() != nil {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "WARNING: rootless Caddy needs net.ipv4.ip_unprivileged_port_start=0")
		fmt.Fprintln(os.Stderr, "to bind ports 80/443. Run 'sudo bakery setup', or manually:")
		fmt.Fprintf(os.Stderr, "  echo '%s' | sudo tee %s\n", unprivilegedPortsSysctlLine, unprivilegedPortsSysctlFile)
		fmt.Fprintln(os.Stderr, "  sudo sysctl --system")
		return
	}

	fmt.Println("Allowing rootless binding to ports 80/443 (net.ipv4.ip_unprivileged_port_start=0)")
	writeCmd := exec.Command("sudo", "tee", unprivilegedPortsSysctlFile)
	writeCmd.Stdin = strings.NewReader(unprivilegedPortsSysctlLine + "\n")
	_ = writeCmd.Run()
	_ = exec.Command("sudo", "sysctl", "--system").Run()
}
