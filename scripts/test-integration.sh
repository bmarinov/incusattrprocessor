#!/usr/bin/env bash
# Runs integration tests against a VM running Incus.
#
# Usage:
#   ./scripts/test-integration.sh [go test flags]
set -euo pipefail

INCUS_SOCKET="${INCUS_SOCKET:-/tmp/incus-test.sock}"
export INCUS_SOCKET

dir="$(cd "$(dirname "$0")/.." && pwd)"

trap '"${dir}/scripts/stop-incus-vm.sh"' EXIT

"${dir}/scripts/start-incus-vm.sh"

# Retrieve the server cert and forward the HTTPS port; 
# sets INCUS_HTTPS_URL, INCUS_CLIENT_CERT, INCUS_CLIENT_KEY, INCUS_SERVER_CERT:
eval "$("${dir}/scripts/setup-incus-https.sh")"

# TODO: drop -p 1. should not be necessary after reconn cleanup:
go test -p 1 "$@" ./...
