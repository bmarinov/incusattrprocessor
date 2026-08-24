#!/bin/bash
# Boots the built collector on a profiles pipeline with incusattr.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

binary="${BINARY:-${repo_root}/dist/otelcol-incus-ebpf-profiler}"
config="${repo_root}/testdata/smoke-collector.yaml"

if [[ ! -x "${binary}" ]]; then
    echo "no collector binary at ${binary}, run scripts/build-collector.sh first" >&2
    exit 1
fi

log=$(timeout 5s "${binary}" \
    --config "${config}" \
    --feature-gates=service.profilesSupport 2>&1 || true)

if ! grep -q "Ready" <<<"${log}"; then
    echo "${log}" >&2
    echo "Collector did not become ready" >&2
    exit 1
fi

echo "profiles pipeline started"
