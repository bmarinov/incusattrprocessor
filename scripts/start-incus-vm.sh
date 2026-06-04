#!/bin/bash
# Boot a local QEMU VM with Incus for integration testing.
# Related files are under $HOME/.cache/incus-test-vm.
#
# Usage:
#   ./scripts/start-incus-vm.sh
#   INCUS_SOCKET=/tmp/incus-test.sock go test ./...
#   ./scripts/stop-incus-vm.sh

set -euo pipefail

INCUS_SOCKET="${INCUS_SOCKET:-/tmp/incus-test.sock}"

dir="$(cd "$(dirname "$0")/.." && pwd)"
cache="${HOME}/.cache/incus-test-vm"
# TODO: any smaller image eg for CI?
image_url="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
image_cache="${cache}/noble-base.qcow2"

ssh_key="${cache}/id_ed25519"
ssh_port=2299
run="${cache}/run"

# Skipping known_hosts for ephemeral VMs.
ssh_opts=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes -i "${ssh_key}" -p "${ssh_port}")

mkdir -p "${cache}" "${run}"

if [[ ! -f "${ssh_key}" ]]; then
  ssh-keygen -t ed25519 -f "${ssh_key}" -N "" -C "incus-test" >/dev/null
fi
pub_key=$(cat "${ssh_key}.pub")

# Generate a client key+cert for mTLS
client_key="${cache}/client.key"
client_cert="${cache}/client.crt"
if [[ ! -f "${client_key}" || ! -f "${client_cert}" ]]; then
  printf 'Generating Incus client certificate...\n' >&2
  openssl req \
    -newkey ec -pkeyopt ec_paramgen_curve:P-384 \
    -nodes -keyout "${client_key}" \
    -x509 -days 3650 \
    -subj "/CN=incus-test-client" \
    -out "${client_cert}" 2>/dev/null
fi

if [[ ! -f "${image_cache}" ]]; then
  printf 'Downloading VM image...\n' >&2
  curl -L --progress-bar -o "${image_cache}.tmp" "${image_url}"
  mv "${image_cache}.tmp" "${image_cache}"
fi

disk="${run}/disk.qcow2"
rm -f "${disk}"
qemu-img create -f qcow2 -b "${image_cache}" -F qcow2 "${disk}" 10G >/dev/null

preseed_b64=$(base64 -w0 < "${dir}/scripts/incus-init-preseed.yaml")
client_cert_b64=$(base64 -w0 < "${client_cert}")

# cloud-init
cat > "${run}/user-data" <<EOF
#cloud-config
ssh_authorized_keys:
  - ${pub_key}
packages:
  - incus
write_files:
  - path: /tmp/incus-preseed.yaml
    encoding: b64
    content: ${preseed_b64}
  - path: /tmp/incus-client.crt
    encoding: b64
    content: ${client_cert_b64}
runcmd:
  - usermod -aG incus-admin ubuntu
  - incus admin init --preseed < /tmp/incus-preseed.yaml
  - incus config trust add-certificate /tmp/incus-client.crt
  - rm /tmp/incus-preseed.yaml /tmp/incus-client.crt
EOF

printf 'instance-id: incus-test\nlocal-hostname: incus-test\n' > "${run}/meta-data"

mkisofs -quiet -output "${run}/cloud-init.iso" \
  -volid cidata -joliet -rock \
  "${run}/user-data" "${run}/meta-data" 2>/dev/null

# Stale VMs:
pidfile="${run}/qemu.pid"
if [[ -f "${pidfile}" ]]; then
  kill "$(cat "${pidfile}")" 2>/dev/null || true
  rm -f "${pidfile}"
fi
rm -f "${INCUS_SOCKET}" "${run}/ssh-fwd.pid"

# boot:
qemu-system-x86_64 \
  -machine q35,accel=kvm \
  -cpu host \
  -m 2048 -smp 2 \
  -drive "file=${disk},format=qcow2,if=virtio" \
  -drive "file=${run}/cloud-init.iso,media=cdrom" \
  -netdev "user,id=net0,hostfwd=tcp::${ssh_port}-:22" \
  -device virtio-net-pci,netdev=net0 \
  -pidfile "${pidfile}" \
  -display none \
  -daemonize
printf 'VM booting with PID %s ...\n' "$(cat "${pidfile}")" >&2

printf 'Waiting for SSH' >&2
i=0
until ssh "${ssh_opts[@]}" -o ConnectTimeout=2 ubuntu@127.0.0.1 true 2>/dev/null; do
  i=$((i+1)); [[ $i -ge 60 ]] && { printf ' timed out\n' >&2; exit 1; }
  printf '.' >&2
  sleep 3
done
printf ' ready\n' >&2

printf 'Waiting for cloud-init' >&2
ssh "${ssh_opts[@]}" ubuntu@127.0.0.1 "cloud-init status --wait >/dev/null 2>&1; true" 2>/dev/null
printf ' done\n' >&2

# Forward the Incus socket to localhost.
# Keep 'sleep infinity': with -N sshd seems to skip initgroups(3) -> perm err.
rm -f "${INCUS_SOCKET}"
ssh "${ssh_opts[@]}" \
    -L "${INCUS_SOCKET}:/var/lib/incus/unix.socket" \
    ubuntu@127.0.0.1 sleep infinity </dev/null >/dev/null 2>/dev/null &
printf '%s\n' "$!" > "${run}/ssh-fwd.pid"
sleep 1

printf 'Incus socket ready at %s\n' "${INCUS_SOCKET}" >&2
