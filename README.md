# Log load client

This project is a golang application to generate logs and send them to various output destinations in various formats. The app runs as a single executable. If more load is needed, scale the app horizontally.
Example:

```bash
$ ./logger --command=generate
goloader seq - localhost.localdomain.0.00000000000000003505C218B3455F5F - 0000000000 - You're screwed !
goloader seq - localhost.localdomain.0.00000000000000003505C218B3455F5F - 0000000001 - Don’t use beef stew as a computer password. It’s not stroganoff.
goloader seq - localhost.localdomain.0.00000000000000003505C218B3455F5F - 0000000002 - failed to reach the cloud, try again on a rainy day
goloader seq - localhost.localdomain.0.00000000000000003505C218B3455F5F - 0000000003 - successfully launched a car in space
goloader seq - localhost.localdomain.0.00000000000000003505C218B3455F5F - 0000000004 - error while reading floppy disk
goloader seq - localhost.localdomain.0.00000000000000003505C218B3455F5F - 0000000005 - Don’t use beef stew as a computer password. It’s not stroganoff.
```
## Usage

examples using docker image:
`podman run quay.io/openshift-logging/cluster-logging-load-client:latest generate`  - start outputting logs to stdout


examples using local binary:  
`./logger --command=generate` - start outputting logs to stdout  
`./logger --command=generate --url=http://localhost:3100/api/prom/push` - send logs to loki  
`./logger --command=generate --logs-per-second=500` - logs 500 log lines per second (default is one log line per seconds)  

Following flags are available:  

```bash
Flags:
    --log-level string             Log level: debug, info, warning, error (default = error) (default "error")
    --command string               Overwrite to control if logs are generated or queried. Allowed values: generate, query.
    --destination string           Overwrite to control where logs are queried or written to. Allowed values: loki, elasticsearch, stdout, file.
    --file string                  The name of the file to write logs to. Only available for File destinations.
    --url string                   URL of Promtail, LogCLI, or Elasticsearch client.
    --disable-security-check bool  Disable security check in HTTPS client.
    --logs-per-second int          The rate to generate logs. This rate may not always be achievable.
    --log-type string              Overwrite to control the type of logs generated. Allowed values: simple, application, synthetic.
    --log-format string            Overwrite to control the format of logs generated. Allowed values: default, crio (mimic CRIO output), csv, json
    --label-type string            Overwrite to control what labels are included in Loki logs. Allowed values: none, client, client-host
    --synthetic-payload-size int   Overwrite to control size of synthetic log line.
    --tenant string                Loki tenant ID for writing logs.
    --queries-per-minute int       The rate to generate queries. This rate may not always be achievable.
    --query string                 Query to use to get logs from storage.
    --query-range string           Duration of time period to query for logs (Loki only).
```

## Build

To build the app run `make build`  
To build docker image use `make build-image`  
To push docker image use `make push-image`  

## Elasticsearch

### Generate logs to elasticsearch v6

Logger sends logs to elasticsearch using its `bulk` API.
Launch an elasticsearch(v6) container:
```
    make run-es
```

Run logger and with remote type  `elasticsearch`: 
```
    make run-local-es-generate
```

### Generate query requests to elasticsearch v6

```
    make run-local-es-query
```


## Loki

### Generate logs to loki

Launch a loki container:
```
    make run-loki
```

Run logger and set with remote type  `loki`:
```
    make run-local-loki-generate
```

### Generate query requests to loki

```
    make run-local-loki-query
```
