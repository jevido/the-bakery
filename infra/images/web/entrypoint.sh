#!/bin/sh
# Runs Caddy as a child so a stop can drain first. On SIGTERM this script
# (PID 1) creates /tmp/draining, which turns /healthz into 503, waits until
# the proxy has stopped routing here, then stops Caddy. Requests that still
# arrive meanwhile are served. Same scheme as infra/images/api/entrypoint.sh.

rm -f /tmp/draining
caddy run --config /etc/caddy/Caddyfile --adapter caddyfile &
pid=$!

drain() {
	touch /tmp/draining
	sleep "${DRAIN_SECONDS:-8}"
	kill -TERM "$pid" 2>/dev/null
}
trap drain TERM INT

while kill -0 "$pid" 2>/dev/null; do
	wait "$pid"
	status=$?
done
exit "${status:-0}"
