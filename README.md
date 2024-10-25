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
      --log-type string              Overwrite to control the type of logs generated. Allowed values: application, audit, simple, synthetic. (default "simple")
      --logs-per-second int          The rate to generate logs. This rate may not always be achievable. (default 1)
      --queries-per-minute int       The rate to generate queries. This rate may not always be achievable. (default 1)
      --query string                 Query to use to get logs from storage.
      --query-range string           Duration of time period to query for logs (Loki only). (default "1s")
      --synthetic-payload-size int   Overwrite to control size of synthetic log line. (default 100)
      --tenant string                Loki tenant ID for writing logs. (default "test")
      --url string                   URL of Promtail, LogCLI, or Elasticsearch client.
      --use-random-hostname          Ensures that the hostname field is unique by adding a random integer to the end.
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
