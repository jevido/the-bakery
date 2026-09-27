#!/bin/sh
# Runs the API as a child so a stop can drain first. Docker sends SIGTERM to
# this script (PID 1). It tells the API to drain (SIGUSR1: health turns 503
# while requests are still served), waits until the health check has marked
# the container unhealthy and the proxy has stopped routing to it, then sends
# SIGTERM so the API finishes in-flight requests and exits.
#
# DRAIN_SECONDS must stay below the platform's stop timeout (Docker: 10 s).

/app/api "$@" &
pid=$!

drain() {
	kill -USR1 "$pid" 2>/dev/null
	sleep "${DRAIN_SECONDS:-8}"
	kill -TERM "$pid" 2>/dev/null
}
trap drain TERM INT

# wait returns early when a trap runs; keep waiting until the API has exited.
while kill -0 "$pid" 2>/dev/null; do
	wait "$pid"
	status=$?
done
exit "${status:-0}"
