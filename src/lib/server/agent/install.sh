#!/bin/sh
# Root-level install shim: creates the dedicated `bakery` system user and
# installs the `bakery` binary + supporting root-level bits (bash
# completion, logrotate). Everything else (rootless Podman prerequisites,
# per-host enrollment, standing up a brand-new instance) is a `bakery`
# subcommand from here on (setup/join/bootstrap) — this script's only job
# is getting the binary onto a box that doesn't have anything yet.
#
# Usage:
#   curl -fsSL <url>/install.sh | bash                    # fresh install
#   curl -fsSL <url>/install.sh | bash -s -- --update      # replace the binary only
#
# <url> is either a running Bakery instance (enrolling an additional host —
# pass --url=<that-instance> so the binary comes from the exact same build)
# or the fixed GitHub Releases location (bootstrapping a brand-new
# instance, before any control plane exists to serve this route at all).
set -eu

BAKERY_USER="bakery"
BIN_PATH="/usr/local/bin/bakery"
GITHUB_RELEASE_BASE="https://github.com/jevido/the-bakery/releases/latest/download"

BAKERY_URL=""
UPDATE=0

for arg in "$@"; do
	case "$arg" in
		--url=*) BAKERY_URL="${arg#--url=}" ;;
		--update) UPDATE=1 ;;
		*)
			echo "Unknown argument: $arg" >&2
			exit 1
			;;
	esac
done

if [ "$(id -u)" -ne 0 ]; then
	echo "install.sh must be run as root (it creates the $BAKERY_USER system user)" >&2
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
	echo "bakery only supports Linux hosts (detected: $OS)" >&2
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

# ensure_user — skipped entirely in --update mode: a re-run only replaces
# the binary, it must never touch the account's config/keys/lingering
# state that setup/join have since built up.
if [ "$UPDATE" -eq 0 ]; then
	if ! id "$BAKERY_USER" >/dev/null 2>&1; then
		echo "--- Creating $BAKERY_USER system user ---"
		useradd --system --create-home --home-dir "/home/$BAKERY_USER" --shell /bin/bash "$BAKERY_USER"
	else
		# A prior install may have left this account with a nologin shell
		# (e.g. distro defaults for --system users); bakery subcommands
		# re-exec into this account via `runuser`, which works either way,
		# but a real shell is what lets an operator `su - bakery` directly
		# to debug, matching scripts/bootstrap-host.sh's existing user.
		CURRENT_SHELL="$(getent passwd "$BAKERY_USER" | cut -d: -f7)"
		case "$CURRENT_SHELL" in
			*/nologin | */false) chsh -s /bin/bash "$BAKERY_USER" ;;
		esac
	fi

	# Rootless systemd --user units (the bakery daemon, Caddy — both set up
	# by `bakery join`) stop dead the moment this install session ends
	# unless lingering is enabled for the account up front, same as
	# scripts/bootstrap-host.sh already does right after creating this
	# user (Phase 08 task 09 retires that script; this is where the same
	# step now lives).
	loginctl enable-linger "$BAKERY_USER"

	# The unit file itself — installed but deliberately not started: there
	# is no agent.env yet (no token/URL known at install time), so
	# ExecStart would just fail. `bakery join` writes this exact same
	# content again once it has a token to write into agent.env, and is
	# what actually enables/starts it — this just means `systemctl --user
	# status bakery-daemon` shows "loaded" rather than "not found"
	# immediately after install, matching jevido/bakery-agent's install.sh
	# shape. Keep in sync with agent/cmd_join.go's installDaemonUnit.
	BAKERY_HOME="$(getent passwd "$BAKERY_USER" | cut -d: -f6)"
	UNIT_DIR="$BAKERY_HOME/.config/systemd/user"
	mkdir -p "$UNIT_DIR" "$BAKERY_HOME/.config/bakery"
	# mkdir -p leaves intermediate dirs (.config, .config/systemd) root-owned
	# since only the leaf paths are named above — bakery join later needs to
	# create its own siblings under .config/ (containers/systemd for the
	# Caddy Quadlet), so the whole subtree has to come back to bakery, not
	# just the two leaves.
	chown -R "$BAKERY_USER:$BAKERY_USER" "$BAKERY_HOME/.config"
	cat >"$UNIT_DIR/bakery-daemon.service" <<-EOF
		[Unit]
		Description=Bakery Agent
		After=network-online.target
		Wants=network-online.target

		[Service]
		Type=simple
		EnvironmentFile=$BAKERY_HOME/.config/bakery/agent.env
		ExecStart=$BIN_PATH daemon
		Restart=on-failure
		RestartSec=5

		[Install]
		WantedBy=default.target
	EOF
	chown "$BAKERY_USER:$BAKERY_USER" "$UNIT_DIR/bakery-daemon.service"
	runuser -u "$BAKERY_USER" -- env XDG_RUNTIME_DIR="/run/user/$(id -u "$BAKERY_USER")" systemctl --user daemon-reload || true
fi

# Download and atomically install the binary. When --url points at a
# running instance, its own build is authoritative (guarantees control
# plane/agent version match); otherwise fall back to the fixed GitHub
# Releases location, since bootstrapping a brand-new instance means
# nothing is running yet to serve /releases/ from.
if [ -n "$BAKERY_URL" ]; then
	RELEASE_BASE="${BAKERY_URL%/}/releases"
else
	RELEASE_BASE="$GITHUB_RELEASE_BASE"
fi
RELEASE_URL="$RELEASE_BASE/bakery-linux-$ARCH"

echo "Downloading bakery from $RELEASE_URL"
# Same directory as the final path so the mv below is a same-filesystem
# rename — atomic, no window where $BIN_PATH is a partially-written file.
TMP_BIN="$(mktemp /usr/local/bin/.bakery-install-XXXXXX)"
if ! curl -fsSL "$RELEASE_URL" -o "$TMP_BIN"; then
	echo "Failed to download bakery binary from $RELEASE_URL" >&2
	rm -f "$TMP_BIN"
	exit 1
fi
chmod 755 "$TMP_BIN"
mv "$TMP_BIN" "$BIN_PATH"

# Bash completion — best-effort only; a missing bash-completion package
# just means no tab-completion, never a failed install.
COMPLETION_DIR=""
if [ -d /usr/share/bash-completion/completions ]; then
	COMPLETION_DIR="/usr/share/bash-completion/completions"
elif [ -d /etc/bash_completion.d ]; then
	COMPLETION_DIR="/etc/bash_completion.d"
fi
if [ -n "$COMPLETION_DIR" ]; then
	cat >"$COMPLETION_DIR/bakery" <<-'EOF'
		_bakery_completions() {
			local cur subcommands
			cur="${COMP_WORDS[COMP_CWORD]}"
			subcommands="daemon setup join bootstrap"
			if [ "$COMP_CWORD" -eq 1 ]; then
				COMPREPLY=($(compgen -W "$subcommands" -- "$cur"))
			fi
		}
		complete -F _bakery_completions bakery
	EOF
fi

# logrotate — nothing under /var/log/bakery/*.log is written yet (the
# daemon logs to the systemd journal, rotated by journald itself), but
# every other root-level piece of state this install creates is set up
# up front rather than added piecemeal later, so this is here in case a
# future component (build-worker, bootstrap) ever does write real log
# files there.
if [ -d /etc/logrotate.d ]; then
	mkdir -p /var/log/bakery
	chown "$BAKERY_USER:$BAKERY_USER" /var/log/bakery
	cat >/etc/logrotate.d/bakery <<-EOF
		/var/log/bakery/*.log {
			weekly
			rotate 4
			compress
			missingok
			notifempty
			su $BAKERY_USER $BAKERY_USER
		}
	EOF
fi

if [ "$UPDATE" -eq 1 ]; then
	echo ""
	echo "bakery updated: $BIN_PATH"
	exit 0
fi

echo ""
echo "bakery installed: $BIN_PATH"
echo ""
echo "Next steps (as root):"
echo "  bakery setup"
echo ""
echo "Then either stand up a brand-new instance on this box:"
echo "  bakery bootstrap --domain=<domain>"
echo "or enroll this host against an already-running instance:"
echo "  bakery join --token=<token> --url=<bakery-url>"
