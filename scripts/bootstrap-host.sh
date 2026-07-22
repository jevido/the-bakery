#!/usr/bin/env bash
# Prepares a fresh Debian 13 box just enough for Bakery's own deployment
# engine to take over: rootless Podman + prerequisites, the `bakery` deploy
# user, firewall, the privileged-port sysctl, and a directly-provisioned
# Postgres. Deliberately does NOT install bakery-agent, the registry, the
# control-plane app, or the build-worker — those are dogfooded through the
# real deployment engine by scripts/self-host-provision.ts (Phase 08 task 10),
# not set up by hand here. See Phase 08 task 09 in the planning repo for the
# full design writeup.
#
# Not idempotent — run once against a pristine box.
# Usage: sudo bash scripts/bootstrap-host.sh
set -euo pipefail

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------

if [ "$(id -u)" -ne 0 ]; then
	echo "Must run as root: sudo bash $0" >&2
	exit 1
fi

if ! grep -q '^VERSION_ID="13"' /etc/os-release 2>/dev/null; then
	echo "This script targets Debian 13 specifically (found: $(grep '^PRETTY_NAME' /etc/os-release 2>/dev/null || echo unknown))." >&2
	exit 1
fi

if id bakery >/dev/null 2>&1 || [ -d /opt/bakery ] || [ -d /home/bakery/bakery ]; then
	echo "This box already has a 'bakery' user or prior bootstrap state — looks like it's" >&2
	echo "already (partially) bootstrapped. This script only supports a pristine box;" >&2
	echo "re-image it and re-run, rather than trying to reconcile existing state." >&2
	exit 1
fi

echo "=== The Bakery — host bootstrap (OS + Postgres only) ==="
echo

read -rp "Control plane domain (e.g. jevido.app): " CONTROL_PLANE_DOMAIN
if [ -z "$CONTROL_PLANE_DOMAIN" ]; then
	echo "A domain is required." >&2
	exit 1
fi

echo
echo "Checking DNS for $CONTROL_PLANE_DOMAIN resolves to this box before going any further..."
BOX_IP="$(curl -4 -fsS https://ifconfig.me)"
RESOLVED_IP="$(getent ahostsv4 "$CONTROL_PLANE_DOMAIN" | awk '{print $1}' | head -n1 || true)"
if [ -z "$RESOLVED_IP" ]; then
	echo "ERROR: $CONTROL_PLANE_DOMAIN does not resolve to anything yet." >&2
	echo "Point its DNS A record at $BOX_IP and re-run." >&2
	exit 1
fi
if [ "$RESOLVED_IP" != "$BOX_IP" ]; then
	echo "ERROR: $CONTROL_PLANE_DOMAIN resolves to $RESOLVED_IP, but this box is $BOX_IP." >&2
	echo "Fix DNS before re-running — the agent's own Caddy can't get a real cert otherwise." >&2
	exit 1
fi
echo "DNS OK: $CONTROL_PLANE_DOMAIN -> $BOX_IP"
echo

echo "Paste the public SSH key that should have access as the 'bakery' deploy" \
	"user (e.g. the contents of hetzner_growth.pub), then press Enter:"
read -r BAKERY_AUTHORIZED_KEY
if [ -z "$BAKERY_AUTHORIZED_KEY" ]; then
	echo "A public key is required so you can still SSH in afterward." >&2
	exit 1
fi

# ---------------------------------------------------------------------------
# OS packages — deliberately minimal. No git/build-essential/unzip/Bun: this
# script never builds or runs application source. The control-plane app,
# registry, and (eventually) build-worker are dogfooded as real Bakery
# deployments (task 10/11), not hand-run here.
# ---------------------------------------------------------------------------

echo "--- Installing OS packages ---"
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y \
	podman uidmap slirp4netns fuse-overlayfs passt aardvark-dns \
	ufw curl ca-certificates openssl

# ---------------------------------------------------------------------------
# Deploy user
# ---------------------------------------------------------------------------

echo "--- Creating bakery deploy user ---"
useradd -m -s /bin/bash bakery
mkdir -p /home/bakery/.ssh
echo "$BAKERY_AUTHORIZED_KEY" >/home/bakery/.ssh/authorized_keys
chmod 700 /home/bakery/.ssh
chmod 600 /home/bakery/.ssh/authorized_keys
chown -R bakery:bakery /home/bakery/.ssh

loginctl enable-linger bakery

# ---------------------------------------------------------------------------
# Rootless Podman needs this later for the agent's own Caddy Quadlet to bind
# ports <1024 (80/443) — same fix src/lib/server/agent/install.sh applies
# itself, just done proactively here since this script already has root and
# a `curl | sh` install (like install.sh's own) may have no sudo access at
# all by the time it would otherwise need to set this.
# ---------------------------------------------------------------------------

echo "--- Allowing rootless binding to ports 80/443 ---"
echo "net.ipv4.ip_unprivileged_port_start=0" >/etc/sysctl.d/50-bakery-unprivileged-ports.conf
sysctl --system >/dev/null

# ---------------------------------------------------------------------------
# Firewall — only 22/80/443. Postgres stays loopback-only, never published
# beyond 127.0.0.1.
# ---------------------------------------------------------------------------

echo "--- Configuring firewall (ufw) ---"
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# ---------------------------------------------------------------------------
# Everything else runs as the unprivileged bakery user. Handed off as a
# generated phase-2 script (rather than trying to export bash functions
# across `runuser`) so the generated Postgres password never appears in this
# process's own argv/env, and is deleted immediately after running.
# ---------------------------------------------------------------------------

mkdir -p /home/bakery/bakery
chown bakery:bakery /home/bakery/bakery

PHASE2=/home/bakery/bakery/.bootstrap-phase2.sh
install -m 700 -o bakery -g bakery /dev/null "$PHASE2"

cat >"$PHASE2" <<PHASE2_EOF
#!/usr/bin/env bash
set -euo pipefail

CONTROL_PLANE_DOMAIN="$CONTROL_PLANE_DOMAIN"

# Non-interactive shells (this one, invoked via \`runuser\`) don't reliably
# inherit the XDG_RUNTIME_DIR/D-Bus session pam_systemd sets up for a real
# login — required for \`systemctl --user\` and rootless Podman to find the
# right runtime dir, even with lingering already enabled.
export XDG_RUNTIME_DIR="/run/user/\$(id -u)"
export DBUS_SESSION_BUS_ADDRESS="unix:path=\${XDG_RUNTIME_DIR}/bus"

HOME_DIR="\$HOME"
BAKERY_DIR="\$HOME_DIR/bakery"
QUADLET_DIR="\$HOME_DIR/.config/containers/systemd"

mkdir -p "\$QUADLET_DIR" "\$BAKERY_DIR/pgdata"

# --- Generate the Postgres password ---
POSTGRES_PASSWORD="\$(openssl rand -hex 24)"

# --- Quadlet: Postgres (foundational infra — everything else is dogfooded
# through the real deployment engine by scripts/self-host-provision.ts,
# Phase 08 task 10, not set up directly here) ---
cat >"\$QUADLET_DIR/db.container" <<UNIT_EOF
[Unit]
Description=Bakery Postgres

[Container]
Image=docker.io/library/postgres:17
Environment=POSTGRES_USER=root
Environment=POSTGRES_DB=local
Environment=POSTGRES_PASSWORD=\${POSTGRES_PASSWORD}
Volume=\$BAKERY_DIR/pgdata:/var/lib/postgresql:Z
PublishPort=127.0.0.1:5434:5432

[Service]
Restart=always

[Install]
WantedBy=default.target
UNIT_EOF

# \`start\`, not \`enable --now\`: Quadlet-generated units are generator-managed
# (loaded from ~/.config/containers/systemd/*.container, regenerated on every
# daemon-reload), and this Podman/systemd combination (5.4.2 on Debian 13)
# rejects \`systemctl --user enable\` on them outright ("is transient or
# generated") — boot-time activation already comes from the unit's own
# [Install] WantedBy=default.target, which the generator wires up itself; a
# separate enable step isn't valid or needed for these.
systemctl --user daemon-reload
systemctl --user start db.service

echo "Waiting for Postgres to accept connections..."
for _ in \$(seq 1 30); do
	if podman exec systemd-db pg_isready -U root >/dev/null 2>&1; then
		break
	fi
	sleep 2
done
if ! podman exec systemd-db pg_isready -U root >/dev/null 2>&1; then
	echo "ERROR: Postgres never became ready." >&2
	exit 1
fi
echo "Postgres is ready."

# --- Write provision.env for task 10's one-off script to consume. Only what
# it needs to connect and start creating rows — everything the eventual
# control-plane deployment itself needs (BETTER_AUTH_SECRET, ENCRYPTION_KEY,
# GitHub credentials, registry credentials) is generated/collected by that
# script as the deployed app's own env vars, not owned by this infra-only
# step. ---
cat >"\$BAKERY_DIR/provision.env" <<ENV_EOF
DATABASE_URL=postgres://root:\${POSTGRES_PASSWORD}@127.0.0.1:5434/local
CONTROL_PLANE_DOMAIN=\${CONTROL_PLANE_DOMAIN}
ENV_EOF
chmod 600 "\$BAKERY_DIR/provision.env"

echo
echo "=== Done ==="
echo "Postgres is running (systemd --user unit: db.service)."
echo "Next: run scripts/self-host-provision.ts (Phase 08 task 10) to issue this"
echo "box's host token, install bakery-agent, and dogfood-deploy the control"
echo "plane + registry through Bakery's own real deployment engine."
PHASE2_EOF

chown bakery:bakery "$PHASE2"
runuser -u bakery -- bash "$PHASE2"
rm -f "$PHASE2"

echo
echo "Bootstrap complete: OS prerequisites + Postgres are ready."
