# Pyroscope backend example

A local setup for visualising the profiles produced by the `incusattr` processor. 

It runs Grafana Pyroscope (OTLP profiles ingest) and optionally Grafana with a preconfigured datasource.

## 1. Start the backend

From this directory:
```bash
docker compose up -d
docker compose --profile grafana up -d # also start Grafana on :3000
```

- Pyroscope UI / OTLP ingest: <http://localhost:4040>
- Grafana (anonymous admin, no login): <http://localhost:3000> with datasource expecting `http://pyroscope:4040`


```bash
until [ "$(curl -s -o /dev/null -w '%{http_code}' http://localhost:4040/ready)" = "200" ]; do
  printf '.'; sleep 5
done
echo " pyroscope ready"
```

> The keepalive settings in `pyroscope.yaml` (`grpc_server_ping_without_stream_allowed`)
> are required. Otherwise `/ready` reports `503`.

## 2. Run the collector

`./collector.yaml` is an example config for the eBPF-profiler built from this repo (see `scripts/build-collector.sh`). 

It must run on the host with Incus processes to profile:

```bash
sudo ./otelcol-incus-ebpf-profiler \
  --config=collector.yaml \
  --feature-gates=service.profilesSupport
```

Pipeline: 
- `profiling` receiver for CPU profiles
- `incusattr` adds `incus.instance.{name,project,location}` attributes;
- `resource` processor manually remaps to **underscore** names (valid in prometheus);

> Pyroscope stores OTLP attribute keys as Prometheus-style labels and does not manually dedot.
> OTLP attributes e.g. `incus.instance.name` are not valid label names in this context.

## 3. Remote collector (optional)

If the collector runs on another host, open a reverse tunnel so its
`http://localhost:4040` endpoint reaches this stack, then run the collector there:

```bash
./forward-pyroscope.sh [remote-host]   # default: incus-host
```

## 4. View profiles

In Grafana -> Explore -> Pyroscope:
- filter by `service_name` (the process) and
`incus_instance_name` (the container)
- e.g. `{incus_instance_name="pg-test"}`.
