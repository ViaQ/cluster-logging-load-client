# Log load client

This project is a golang application to generate logs and send them to various output destinations in various formats.
The app runs as a single executable. Per configuration can spawn multiple threads. If more load is needed, scale the app horizontally.
Example:

```bash
$ ./logger generate
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
`./logger generate` - start outputting logs to stdout  
`./logger generate --url=http://localhost:3100/api/prom/push` - send logs to loki  
`./logger generate ---log-lines-rate=500` - logs 500 log lines per second (default is one log line per seconds)  

Following flags are available:  

```bash
Flags:
      --config string                config file (default is $HOME/logger.yaml)
      --destination string           Log Destination: loki, elasticsearch, stdout, file. (default stdout) (default "stdout")
      --destination-url string       send logs via api using the provided url (e.g http://localhost:3100/api/prom/push)
      --file string                  The file to output (default: output) (default "output")
  -h, --help                         help for logger
      --log-level string             Log level: debug, info, warning, error (default = error) (default "error")
      --log-lines-rate int           The total amount of log lines per thread per second to generate.(default 1) (default 1)
      --loki-labels string           Loki labels: none,host,random (default = random) (default "random")
      --loki-tenant-ID string        Loki tenantID (default = fake)
      --output-format string         The output format: default, crio (mimic CRIO output), csv, json (default "default")
      --queries stringArray          list of queries e.g. {client="promtail"} (default = none)
      --query-file string            Query file name (default = none)
      --source string                Log lines Source: simple, application, synthetic. (default simple) (default "simple")
      --synthetic-payload-size int   Payload length [int] (default = 100) (default 100)
      --tenant string                Alt. Loki tenantID flag (see --loki-tenant-ID) (default "fake")
      --threads int                  Number of threads.(default 1) (default 1)
      --totalLogLines int            Total number of log lines per thread (default 0 - infinite)
      --url string                   Alt. destination flag (see --destination-url)
```

Environment variables are supported using prefix "LOADCLIENT" - examples: 

`LOADCLIENT_LOG_LEVEL=DEBUG ./logger generate`  
`podman run -e LOADCLIENT_LOG_LEVEL=DEBUG quay.io/openshift-logging/cluster-logging-load-client:latest generate`  

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
