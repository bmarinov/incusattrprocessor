#!/bin/bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

collector_version="0.151.0"
ocb_module="go.opentelemetry.io/collector/cmd/builder"
ocb_bin="${repo_root}/bin/ocb"
manifest="${repo_root}/otelcol-builder.yaml"
build_dir="${repo_root}/_build"
dist_dir="${repo_root}/dist"
GOARCH="${GOARCH:-$(go env GOARCH)}"
GOOS="${GOOS:-linux}"

install_ocb() {
    local installed_version
    if [[ -x "${ocb_bin}" ]]; then
        installed_version=$("${ocb_bin}" version 2>&1 | grep -oP '\d+\.\d+\.\d+' | head -1 || true)
        if [[ "${installed_version}" == "${collector_version}" ]]; then
            return
        fi
        echo "found outdated ocb version ${installed_version}"
    fi
    echo "Installing ocb v${collector_version}..."
    GOBIN="${repo_root}/bin" go install "${ocb_module}@v${collector_version}"
    mv "${repo_root}/bin/builder" "${ocb_bin}"
}

install_ocb

echo "generating otelcol-incus-ebpf-profiler v${collector_version}..."
CGO_ENABLED=0 "${ocb_bin}" \
    --config="${manifest}" \
    --skip-get-modules=false \
    --skip-compilation=true

# ocb guesses an invalid  package name:
sed -i \
    -e 's|otelcol-processor-incus "github\.com/bmarinov/otelcol-processor-incus"|incusattrprocessor "github.com/bmarinov/otelcol-processor-incus"|g' \
    -e 's|otelcol-processor-incus\.NewFactory|incusattrprocessor.NewFactory|g' \
    "${build_dir}/components.go"

(cd "${build_dir}" && go mod tidy)

echo "compiling ${GOOS}/${GOARCH}..."
(cd "${build_dir}" && CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
    -trimpath \
    -o "otelcol-incus-ebpf-profiler" \
    -ldflags="-s -w" \
    -tags grpcnotrace \
    .)

mkdir -p "${dist_dir}"
cp "${build_dir}/otelcol-incus-ebpf-profiler" "${dist_dir}/"

echo "binary: ${dist_dir}/otelcol-incus-ebpf-profiler"
