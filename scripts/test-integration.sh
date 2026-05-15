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
INCUS_SOCKET="${INCUS_SOCKET}" go test "$@" ./...
