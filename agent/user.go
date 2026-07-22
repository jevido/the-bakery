package main

// bakeryUsername is the dedicated system user all Bakery state (rootless
// Podman, Quadlet units, Postgres, Caddy, the daemon itself) runs as —
// created by the root-run install shim, replacing the old "runs as
// whichever account you SSH'd in as" model.
const bakeryUsername = "bakery"
