#!/bin/sh
# Installs bakery-agent as a rootless systemd --user service.
# Usage: curl -fsSL <bakery-url>/install.sh | sh -s -- --token=<token> --url=<bakery-url>
set -eu

TOKEN=""
BAKERY_URL=""

for arg in "$@"; do
	case "$arg" in
		--token=*) TOKEN="${arg#--token=}" ;;
		--url=*) BAKERY_URL="${arg#--url=}" ;;
		*)
			echo "Unknown argument: $arg" >&2
			exit 1
			;;
	esac
done

if [ -z "$TOKEN" ] || [ -z "$BAKERY_URL" ]; then
	echo "Usage: install.sh --token=<token> --url=<bakery-url>" >&2
	exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
	echo "curl is required but was not found on PATH" >&2
	exit 1
fi
if ! command -v systemctl >/dev/null 2>&1; then
	echo "systemctl is required but was not found on PATH (systemd is required)" >&2
	exit 1
fi

OS="$(uname -s)"
if [ "$OS" != "Linux" ]; then
	echo "bakery-agent only supports Linux hosts (detected: $OS)" >&2
	exit 1
fi

ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
	x86_64) ARCH="amd64" ;;
	aarch64 | arm64) ARCH="arm64" ;;
	*)
		echo "Unsupported architecture: $ARCH_RAW" >&2
		exit 1
		;;
esac

BIN_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/bakery"
UNIT_DIR="$HOME/.config/systemd/user"
BIN_PATH="$BIN_DIR/bakery-agent"
ENV_PATH="$CONFIG_DIR/agent.env"
UNIT_PATH="$UNIT_DIR/bakery-agent.service"

mkdir -p "$BIN_DIR" "$CONFIG_DIR" "$UNIT_DIR"

RELEASE_URL="$BAKERY_URL/releases/bakery-agent-linux-$ARCH"
echo "Downloading agent from $RELEASE_URL"
TMP_BIN="$(mktemp)"
if ! curl -fsSL "$RELEASE_URL" -o "$TMP_BIN"; then
	echo "Failed to download agent binary from $RELEASE_URL" >&2
	rm -f "$TMP_BIN"
	exit 1
fi
chmod +x "$TMP_BIN"
mv "$TMP_BIN" "$BIN_PATH"

(
	umask 077
	cat >"$ENV_PATH" <<-EOF
		BAKERY_TOKEN=$TOKEN
		BAKERY_URL=$BAKERY_URL
	EOF
)
chmod 600 "$ENV_PATH"

cat >"$UNIT_PATH" <<-EOF
	[Unit]
	Description=Bakery Agent
	After=network-online.target
	Wants=network-online.target

	[Service]
	Type=simple
	EnvironmentFile=$ENV_PATH
	ExecStart=$BIN_PATH
	Restart=on-failure
	RestartSec=5

	[Install]
	WantedBy=default.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now bakery-agent

echo ""
echo "bakery-agent installed and started."
echo "  Status: systemctl --user status bakery-agent"
echo "  Logs:   journalctl --user -u bakery-agent -f"
echo ""
echo "Rootless systemd --user services stop when you fully log out unless"
echo "lingering is enabled for this account. If this host doesn't stay"
echo "logged in, run (as a user with sudo):"
echo "  sudo loginctl enable-linger $(id -un)"

# Bakery manages its own per-host reverse proxy (Caddy) as a second Quadlet-
# generated unit, installed here rather than pushed as an agent command
# (Phase 05 task 01): it's fixed infrastructure with no per-deployment
# identity, so it doesn't fit hostCommand's per-deployment payload model the
# way app deploys (Phase 04) do, and it needs to exist before the first app
# deploy can rely on it.
QUADLET_DIR="$HOME/.config/containers/systemd"
CADDY_CONFIG_DIR="$HOME/.config/bakery/caddy"
CADDY_DATA_DIR="$HOME/.local/share/bakery/caddy"
CADDYFILE="$CADDY_CONFIG_DIR/Caddyfile"
CADDY_UNIT_PATH="$QUADLET_DIR/caddy.container"

mkdir -p "$QUADLET_DIR" "$CADDY_CONFIG_DIR" "$CADDY_DATA_DIR"

# Only seed a default Caddyfile if one doesn't exist yet — re-running this
# script (e.g. to reinstall the agent) must not clobber site config that
# later tasks (02-04) have since written for live domains.
if [ ! -f "$CADDYFILE" ]; then
	cat >"$CADDYFILE" <<-EOF
		# Managed by bakery-agent. Site blocks for deployed apps and custom
		# domains are added here automatically — avoid hand-editing.
	EOF
fi

cat >"$CADDY_UNIT_PATH" <<-EOF
	[Unit]
	Description=Bakery Reverse Proxy (Caddy)
	After=network-online.target

	[Container]
	Image=docker.io/library/caddy:2
	PublishPort=80:80
	PublishPort=443:443
	Volume=$CADDY_CONFIG_DIR:/etc/caddy:Z
	Volume=$CADDY_DATA_DIR:/data:Z
	AutoUpdate=registry

	[Service]
	Restart=always
	TimeoutStartSec=90

	[Install]
	WantedBy=default.target
EOF

# Rootless Podman can't bind ports <1024 unless the host allows it. Best
# effort only: a `curl | sh` install may have no sudo access at all, and
# that must not fail the agent install that already succeeded above.
if command -v sysctl >/dev/null 2>&1; then
	CURRENT_UNPRIVILEGED_START="$(sysctl -n net.ipv4.ip_unprivileged_port_start 2>/dev/null || echo 1024)"
	if [ "${CURRENT_UNPRIVILEGED_START:-1024}" -gt 80 ] 2>/dev/null; then
		if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
			echo "Allowing rootless binding to ports 80/443 (net.ipv4.ip_unprivileged_port_start=0)"
			echo "net.ipv4.ip_unprivileged_port_start=0" | sudo tee /etc/sysctl.d/50-bakery-unprivileged-ports.conf >/dev/null
			sudo sysctl --system >/dev/null
		else
			echo "" >&2
			echo "WARNING: rootless Caddy needs net.ipv4.ip_unprivileged_port_start=0" >&2
			echo "to bind ports 80/443. Run (as a user with sudo):" >&2
			echo "  echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee /etc/sysctl.d/50-bakery-unprivileged-ports.conf" >&2
			echo "  sudo sysctl --system" >&2
		fi
	fi
fi

systemctl --user daemon-reload
if systemctl --user enable --now caddy.service; then
	echo ""
	echo "bakery-managed Caddy reverse proxy installed and started."
	echo "  Status: systemctl --user status caddy"
	echo "  Logs:   journalctl --user -u caddy -f"
else
	echo "" >&2
	echo "WARNING: Caddy unit failed to start — check 'journalctl --user -u caddy'." >&2
	echo "This does not affect bakery-agent, which is already running." >&2
fi
