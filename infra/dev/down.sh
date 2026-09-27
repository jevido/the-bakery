#!/usr/bin/env bash
# Stops every local dev server this repo runs, found by the ports they listen
# on (passed as arguments). The whole process group behind each listener is
# stopped, so `task` and the processes it spawned go down together. Dev tools
# that put a listener in a group of its own (wails3 dev runs Vite that way)
# are found by walking up from the listener through known dev-tool parents,
# and their groups are stopped too.
set -u

# Parents that are part of a dev server, never a shell or terminal.
DEV_TOOLS=" task wails3 go bun node air "

self=$(ps -o pgid= -p $$ | tr -d ' ')
stopped=0

listeners() {
  if command -v lsof >/dev/null 2>&1; then
    lsof -tiTCP:"$1" -sTCP:LISTEN 2>/dev/null
  else
    ss -ltnpH "sport = :$1" 2>/dev/null | grep -o 'pid=[0-9]*' | cut -d= -f2
  fi
}

# Prints the process groups of pid and of every dev-tool ancestor above it.
groups_of() {
  local p=$1
  while [ -n "$p" ] && [ "$p" -gt 1 ]; do
    ps -o pgid= -p "$p" 2>/dev/null | tr -d ' '
    p=$(ps -o ppid= -p "$p" 2>/dev/null | tr -d ' ')
    [ -n "$p" ] || break
    case "$DEV_TOOLS" in *" $(ps -o comm= -p "$p" 2>/dev/null) "*) ;; *) break ;; esac
  done
}

for port in "$@"; do
  for pid in $(listeners "$port" | sort -u); do
    name=$(ps -o comm= -p "$pid" 2>/dev/null || echo "pid $pid")
    signalled=0
    for pgid in $(groups_of "$pid" | sort -u); do
      # Never signal our own group or init's.
      if [ "$pgid" -gt 1 ] && [ "$pgid" != "$self" ]; then
        kill -TERM -- "-$pgid" 2>/dev/null && signalled=1
      fi
    done
    [ "$signalled" = 1 ] || kill -TERM "$pid" 2>/dev/null
    echo "stopped :$port ($name)"
    stopped=1
  done
done

[ "$stopped" = 1 ] || echo "nothing running on: ${*:-(no ports configured)}"
