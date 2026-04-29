# otelcol-processor-incusattr

## Build

### OCB distribution

Bundle in a custom collector binary. Includes the ebpf-profiler, see [otelcol-builder.yaml](./otelcol-builder.yaml) for details.

```sh
# build for the host arch:
./scripts/build-collector.sh

# cross-compile (e.g. arm64):
GOARCH=arm64 ./scripts/build-collector.sh
```
