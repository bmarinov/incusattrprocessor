# incusattrprocessor

OTEL profiles processor for [Incus](https://linuxcontainers.org/incus/) container metadata.

For each profiled process it reads `/proc/<pid>/cgroup`, extracts the container name from the `lxc.payload.<name>` cgroup path, and attaches the following attributes to the resource:
- `incus.instance.name`
- `incus.instance.project`
- `incus.instance.location`

_Note:_ The eBPF profiler does not have visibility into processes running on VMs. Only Incus containers (LXC-backed) are supported.

## Project status

The processor is in early preview/alpha. OTEL profiles are in alpha.

Note: run collector with `--feature-gates=service.profilesSupport` to enable profiling.

## Prerequisites
- Host running Incus with LXC containers
- [opentelemetry-ebpf-profiler](https://github.com/open-telemetry/opentelemetry-ebpf-profiler) to collect samples upstream
- Supported [Linux kernel](https://github.com/open-telemetry/opentelemetry-ebpf-profiler#supported-linux-kernel-version) version for the eBPF profiler

## Configuration

Currently only the Incus connection can be configured. Unix socket and HTTPS are mutually exclusive. Configure exactly one of `connection.socket_path` and `connection.https`.

By default the client assumes the collector has permissions to access the unix socket on the Incus host.

Effective default configuration:
```yaml
processors:
  incusattr:
    connection:
      socket_path: /var/lib/incus/unix.socket
```

Connect over HTTPS with TLS certs:
```yaml
processors:
  incusattr:
    connection:
      https:
        url: https://incus.local:8443
        client_cert: /etc/incus/client.crt
        client_key: /etc/incus/client.key
        server_cert: /etc/incus/server.crt
```


## Usage

Add the processor to the [OCB](https://opentelemetry.io/docs/collector/extend/ocb/) builder manifest, pinned to a released tag:

```yaml
processors:
  - gomod: github.com/bmarinov/incusattrprocessor v0.1.0
```


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

Benchmark results are in [BENCHMARKS.md](./BENCHMARKS.md); regenerate with `scripts/bench.sh`.


### Integration tests

Running `./scripts/test-integration.sh` will set up a VM with Incus and run all tests, including the client integration suite.

The tests require `INCUS_SOCKET` to be set.  
The HTTPS client tests also require the following env vars:
- `INCUS_HTTPS_URL`
- `INCUS_HTTPS_CLIENT_CERT`
- `INCUS_HTTPS_CLIENT_KEY`
- `INCUS_HTTPS_SERVER_CERT`

To run and debug the tests in VSCode:
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

# optional (HTTPS tests) - set env vars according to script output:
# ./scripts/setup-incus-https.sh

# run / debug integration tests in VS Code

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

## License

[Apache License 2.0](./LICENSE).
