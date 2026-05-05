# otelcol-processor-incusattr

OTEL profiles processor for [Incus](https://linuxcontainers.org/incus/) container metadata. 

Attributes such as `incus.instance.name` and `incus.instance.location` are added to events for PIDs running in LXC containers.

## Build

### OCB distribution

Bundle in a custom collector binary. Includes the ebpf-profiler, see [otelcol-builder.yaml](./otelcol-builder.yaml) for details.

```sh
# build for the host arch:
./scripts/build-collector.sh

# cross-compile (e.g. arm64):
GOARCH=arm64 ./scripts/build-collector.sh
```

For each profiled process it reads `/proc/<pid>/cgroup`, extracts the container name from the `lxc.payload.<name>` cgroup path, and attaches `incus.instance.name`, `incus.instance.project`, and `incus.instance.location` to the resource.

## Prerequisites

- Go 1.24+
- Linux host running Incus with LXC containers

## Development

```bash
go test ./...
go vet ./...
```


Output: `dist/otelcol-incus-ebpf-profiler`

## Debug run on a remote LXC host

```bash
# deploy binary + config
./scripts/deploy-collector.sh user@host

# run on the remote (requires root for eBPF)
ssh user@host sudo ~/dev/otelcol-incus/otelcol-incus-ebpf-profiler \
  --config=~/dev/otelcol-incus/debug-collector.yaml \
  --feature-gates=service.profilesSupport
```
