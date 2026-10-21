# Contributing

## Updating the OpenTelemetry Collector

The profiles signal is alpha. Its Go APIs and the OTLP profiles proto still change between collector releases. All OpenTelemetry versions in this repo must move together.

### Constraints

- **One collector release everywhere.** Collector releases ship stable modules as `v1.N.0` and unstable as `v0.M.0`. Example: `v1.64.0/v0.158.0` (from the GitHub release). Ensure that every `go.opentelemetry.io/collector/*` module in `go.mod` is on the correct pair.
- **The ebpf-profiler determines collector release.** Target the version used in the `go.opentelemetry.io/ebpf-profiler` build.
- **Match `go.mod` versions in distribution.** Everything in `otelcol-builder.yaml` and `collector_version` in `scripts/build-collector.sh` are on the same release.

`go.opentelemetry.io/otel/*` (the SDK) has a separate release cycle.

### Steps

1. Find the target release:

   ```sh
   go list -m go.opentelemetry.io/ebpf-profiler@latest
   curl -s https://proxy.golang.org/go.opentelemetry.io/ebpf-profiler/@v/<version>.mod \
     | grep -E 'collector/pdata(/pprofile)? '
   ```

   `pdata` shows the `v1` version, `pdata/pprofile` the `v0` one.

2. In `go.mod`: replace the old pair with the new one on every `go.opentelemetry.io/collector/` line; run `go mod tidy`.

3. In `otelcol-builder.yaml`:
   - Update `dist.version`, every component and provider.
   - Update `go.opentelemetry.io/ebpf-profiler` to the version from step (1).
   - set `github.com/bmarinov/incusattrprocessor` to the tag this change will be released as. 


4. Update `collector_version` in `scripts/build-collector.sh`.

5. Verify:

   The new processor tag doesn't exist yet, so uncomment `path: .` under it in `otelcol-builder.yaml` to build. Don't commit it uncommented.

   ```sh
   go build ./... && go vet ./... && go test -race ./...
   ./scripts/build-collector.sh
   ./scripts/smoke-collector.sh
   
   # must print exactly the two target versions
   grep -oE '^\s+go.opentelemetry.io/collector/[^ ]+ v[0-9.]+' _build/go.mod | awk '{print $2}' | sort -u
   ```

5. Run `scripts/bench.sh` and compare with the previous results in [BENCHMARKS.md](BENCHMARKS.md).

6. Open one PR, e.g. `chore(build): update otel collector 0.158`. After merge, tag the release with `make tag TAG=vX.Y.Z`.
