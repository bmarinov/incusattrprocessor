#!/usr/bin/env bash
# Run all benchmarks and optionally summarise with benchstat.
# Output is written to bench.out in the repo root.
#
# Usage: scripts/bench.sh [--save]
#
# Requirements: Go toolchain in PATH; benchstat optional (golang.org/x/perf/cmd/benchstat).
set -Eeuo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

cd "$repo_root"

go_version="$(go version)"
host="$(uname -sr)"

echo "# ${host} / ${go_version}"
echo

go test -run='^$' -bench=. -benchmem -count=6 ./... | tee bench.out

if ! command -v benchstat &>/dev/null; then
    echo "benchstat not found; skipping summary (go install golang.org/x/perf/cmd/benchstat@latest)" >&2
    exit 0
fi

{
    echo "# Benchmarks"
    echo
    echo "${host} / ${go_version}"
    echo
    echo '```'
    benchstat bench.out
    echo '```'
} >BENCHMARKS.md
echo "wrote BENCHMARKS.md"

# Regression summary against the baseline.
if [[ -f bench-baseline.out ]]; then
    echo
    echo "# change vs baseline"
    benchstat bench-baseline.out bench.out
fi

# scripts/bench.sh --save to save as new baseline:
if [[ "${1:-}" == "--save" ]]; then
    cp bench.out bench-baseline.out
    echo "updated bench-baseline.out"
fi
