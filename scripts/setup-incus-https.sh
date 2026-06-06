#!/usr/bin/env bash
# Retrieves the Incus server certificate and forwards the HTTPS port to localhost.
#
# Must run AFTER start-incus-vm.sh.
# 
# env vars printed to stdout, execute with:
#   eval "$(./scripts/setup-incus-https.sh)"
#
# Output:
#   INCUS_HTTPS_URL    https://127.0.0.1:<port>
#   INCUS_CLIENT_CERT  path to client certificate
#   INCUS_CLIENT_KEY   path to client private key
#   INCUS_SERVER_CERT  path to server cert

set -euo pipefail

INCUS_SOCKET="${INCUS_SOCKET:-/tmp/incus-test.sock}"
https_port="${INCUS_HTTPS_PORT:-8443}"

cache="${HOME}/.cache/incus-test-vm"
run="${cache}/run"

ssh_opts=(
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o BatchMode=yes
  -i "${cache}/id_ed25519"
  -p 2299
)

# Retrieve server cert from GET /1.0 on the socket:
server_cert="${cache}/server.crt"
curl --silent --unix-socket "${INCUS_SOCKET}" http://unix/1.0 \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['metadata']['environment']['certificate'])" \
  > "${server_cert}"

# Forward the HTTPS port from inside the VM:
https_fwd_pid="${run}/https-fwd.pid"
if [[ -f "${https_fwd_pid}" ]]; then
  kill "$(cat "${https_fwd_pid}")" 2>/dev/null || true
  rm -f "${https_fwd_pid}"
fi

ssh "${ssh_opts[@]}" \
  -L "${https_port}:localhost:${https_port}" \
  ubuntu@127.0.0.1 sleep infinity </dev/null >/dev/null 2>/dev/null &
printf '%s\n' "$!" > "${https_fwd_pid}"
sleep 1

printf 'Incus HTTPS ready at https://127.0.0.1:%s\n' "${https_port}" >&2

# export statements for eval.
printf 'export INCUS_HTTPS_URL=https://127.0.0.1:%s\n' "${https_port}"
printf 'export INCUS_CLIENT_CERT=%s\n' "${cache}/client.crt"
printf 'export INCUS_CLIENT_KEY=%s\n' "${cache}/client.key"
printf 'export INCUS_SERVER_CERT=%s\n' "${server_cert}"
