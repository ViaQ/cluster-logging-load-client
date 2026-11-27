# Log load client

This project is a golang application to generate logs and send them to various output destinations in various formats. The app runs as a single executable. If more load is needed, scale the app horizontally.

Example:

```shell
$ ./logger
goloader seq - hostname.example.com - 0000000000 - infinite loop succeeded in less than 3 seconds
goloader seq - hostname.example.com - 0000000001 - Don’t use beef stew as a computer password. It’s not stroganoff.
goloader seq - hostname.example.com - 0000000002 - cannot over-write a locked variable.
goloader seq - hostname.example.com - 0000000003 - failed to get an error message
```
## Usage

The following flags are available:

```shell
$ ./logger --help
Usage of ./logger:
      --command string               Overwrite to control if logs are generated or queried. Allowed values: generate, query. (default "generate")
      --destination string           Overwrite to control where logs are queried or written to. Allowed values: loki, elasticsearch, stdout, file. (default "stdout")
      --disable-security-check       Disable security check in HTTPS client.
      --file string                  The name of the file to write logs to. Only available for "File" destinations. (default "output.txt")
      --label-type string            Overwrite to control what labels are included in Loki logs. Allowed values: none, client, client-host (default "none")
      --log-format string            Overwrite to control the format of logs generated. Allowed values: default, crio (mimic CRIO output), csv, json (default "default"), raw
      --log-level string             Overwrite to control the level of logs emitted. Allowed values: debug, info, warning, error (default "error")
      --log-type string              Overwrite to control the type of logs generated. Allowed values: application, audit, simple, synthetic, config. (default "simple")
      --loader-id string             Value substituted for {{LOADER_ID}} in payload templates.
      --logs-per-second int          The rate to generate logs. This rate may not always be achievable. (default 1)
      --payload-config string        Path to a JSON payload config (weighted message templates). Required for --log-type config.
      --queries-per-minute int       The rate to generate queries. This rate may not always be achievable. (default 1)
      --query string                 Query to use to get logs from storage.
      --query-range string           Duration of time period to query for logs (Loki only). (default "1s")
      --sent-count-file string       When set (with --duration), write the total emitted line count to this file when emission ends.
      --start-signal-file string     When set, wait until this file exists before emitting any logs.
      --synthetic-payload-size int   Overwrite to control size of synthetic log line. (default 100)
      --duration duration            Emit logs for this long, then idle forever. 0 means unlimited.
      --tenant string                Loki tenant ID for writing logs. (default "test")
      --url string                   URL of Promtail, LogCLI, or Elasticsearch client.
      --use-random-hostname          Ensures that the hostname field is unique by adding a random integer to the end.
```

## Config-driven payloads and benchmark coordination

For benchmarks that need a controlled message mix and exact loss accounting, use `--log-type config`
with a payload config. Each emitted message is chosen from the templates by weight using a seeded
random generator (so runs are reproducible), and placeholders are substituted per message:

* `{{SEQ}}`: monotonic per-process counter, starting at 0
* `{{LOADER_ID}}`: the value of `--loader-id` (default empty)

```json
{
  "seed": 1,
  "templates": [
    {"weight": 90, "message": "level=info loader={{LOADER_ID}} seq={{SEQ}} request handled"},
    {"weight": 9,  "message": "level=warning loader={{LOADER_ID}} seq={{SEQ}} slow upstream"},
    {"weight": 1,  "message": "level=error loader={{LOADER_ID}} seq={{SEQ}} request failed"}
  ]
}
```

The template text is the complete message; no hostname or component fields are added. Pair it with
`--log-format raw` so the line is written unmodified (when running in a container, kubelet adds the
CRIO wrapper). Weights must be non-negative and sum to more than zero.

Coordination flags:

* `--start-signal-file`: block (polling every 100ms) until the file exists, so several loaders can be released together.
* `--duration`: emit for this long, write `--sent-count-file`, then idle until SIGTERM/SIGINT. The process never
  exits on its own, because a container restart would reset `{{SEQ}}` and invalidate loss accounting.
* `--sent-count-file`: receives the total number of lines emitted. Compared against the sequence numbers
  observed at the destination, it gives the number of lost lines.

```shell
$ ./logger --command generate --destination stdout --log-format raw \
    --logs-per-second 100 --log-type config --payload-config payload.json \
    --loader-id 3 --start-signal-file /tmp/start \
    --duration 10m --sent-count-file /tmp/sent
```

## Docker Image

```shell
podman run --rm -it quay.io/openshift-logging/cluster-logging-load-client:latest
```

## From Source

```shell
# Build binary ("logger")
$ make build
# Default configuration
$ ./logger
# Increased log rate
$ ./logger --logs-per-second=500
# Push logs directly to Loki
$ ./logger --destination=loki --uri=http://localhost:3100/loki/api/v1/push
```

## Build

```shell
# Build the binary
$ make build
# Build the Docker image
$ make build-image
```
