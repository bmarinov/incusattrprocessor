#!/usr/bin/env bash
# Reverse-tunnel the local Pyroscope port to a remote host, so a collector
# running there can reach Pyroscope at localhost:4040 (see collector.yaml).
#
# Usage: ./forward-pyroscope.sh [remote-host]   (default: incus-host)
# ---
set -euo pipefail

remote="${1:-incus-host}"
local_port=4040

cleanup() {
    printf '\nTunnel closed. %s localhost:%s is no longer forwarded.\n' "$remote" "$local_port"
}
trap cleanup EXIT INT

printf 'Pyroscope reverse tunnel: %s:localhost:%s → local:%s\n' "$remote" "$local_port" "$local_port"
printf 'Keep this running while the collector is active. Ctrl+C to stop.\n\n'

ssh -R "${local_port}:localhost:${local_port}" -N "$remote"
