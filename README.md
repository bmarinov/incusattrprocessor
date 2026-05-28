# otelcol-processor-incusattr

OTEL profiles processor for [Incus](https://linuxcontainers.org/incus/) container metadata.

For each profiled process it reads `/proc/<pid>/cgroup`, extracts the container name from the `lxc.payload.<name>` cgroup path, and attaches the following attributes to the resource:
- `incus.instance.name`
- `incus.instance.project`
- `incus.instance.location`

_Note:_ The eBPF profiler does not have visibility into processes running on VMs. Only Incus containers (LXC-backed) are supported.

## Project status

The processor is in early preview/alpha. OTEL profiles are in alpha.

Note: run collector with `--feature-gates=service.profilesSupport` to enable profiling.

## Configuration

Currently only the Incus socket path can be configured.

Effective default configuration:
```yaml
processors:
  incusattr:
    connection:
      socket_path: /var/lib/incus/unix.socket
```

## Prerequisites
- Host running Incus with LXC containers
- Supported [Linux kernel](https://github.com/open-telemetry/opentelemetry-ebpf-profiler#supported-linux-kernel-version) version for the upstream epbf profiler.

## Build

Bundle together with the ebpf-profiler into a custom collector binary via [OCB](https://opentelemetry.io/docs/collector/extend/ocb/).

See [otelcol-builder.yaml](./otelcol-builder.yaml) for details.

```sh
# build for the host arch:
./scripts/build-collector.sh

# cross-compile (e.g. arm64):
GOARCH=arm64 ./scripts/build-collector.sh
```

Output: `dist/otelcol-incus-ebpf-profiler`

## Development

TBD

```bash
go test ./...
go vet ./...
```

The cgroup probe test needs a live LXC host to run.


### Integration tests

Running `./scripts/test-integration.sh` will set up a VM with Incus and run all tests, including the client integration suite.

The tests require `INCUS_SOCKET` to be set.

To run and debug the tests:
`settings.json`:
```json
{
  "go.testEnvVars": {
    "INCUS_SOCKET": "/tmp/incus-test.sock"
  }
}
```

The tests at `./internal/incus/integration_tests/` can now be executed. The VM must be stopped and cleaned up manually:
```sh
# start the VM
./scripts/start-incus-vm.sh

# run / debug integration tests

# clean up
./scripts/stop-incus-vm.sh
```


### Remote debugging

Build `probe.test` + fetch `dlv`, upload both to the Incus host, then run:

```sh
dlv exec --headless --listen=:2345 --api-version=2 \
  ./probe.test -- -test.v -test.run TestProbe_cgroupMetadata
```

Tunnel `:2345` back locally and attach from VS Code.

### Run on a remote Incus host

```bash
# deploy binary + config
./scripts/deploy-collector.sh user@host

# run on the remote (requires root for eBPF)
ssh user@host sudo ~/dev/otelcol-incus/otelcol-incus-ebpf-profiler \
  --config=~/dev/otelcol-incus/debug-collector.yaml \
  --feature-gates=service.profilesSupport
```
