#!/bin/bash
# Stops the VM from start-incus-vm.sh (assumes pid file):
set -euo pipefail

INCUS_SOCKET="${INCUS_SOCKET:-/tmp/incus-test.sock}"
run="${HOME}/.cache/incus-test-vm/run"

kill_pid() {
  local file="$1"
  [[ -f "${file}" ]] || return 0
  kill "$(cat "${file}")" 2>/dev/null || true
  rm -f "${file}"
}

kill_pid "${run}/https-fwd.pid"
kill_pid "${run}/ssh-fwd.pid"
kill_pid "${run}/qemu.pid"
rm -f "${INCUS_SOCKET}"

printf 'Incus VM stopped.\n'
