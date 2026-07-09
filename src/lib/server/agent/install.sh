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
