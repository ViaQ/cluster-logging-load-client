# Log load client

This project is a simple golang application to generate random logs.

This app runs in a single goroutine, if you want to generate more load, scale the app.

To build the app run `make build`.

`./logger generate` to start outputting log to stdout you'll need Promtail or an agent to send the logs.
`./logger generate --url=http://localhost:3100/api/prom/push` to send log to Loki directly

By default the app log 500 logs per seconds, this can be increased or decrease using the flag `--logps`. For example
`./logger generate --logps=1` will log one line per second.

To build a docker image use `make build-image`.

To run the image use: `docker run ctovena/logger:0.1 --url=http://localhost:3100/api/prom/push`

## Generate log to elasticsearch v6

Logger sends logs to elasticsearch using its `bulk` API.
Launch an elasticsearch(v6) container:
```
    make run-es
```

Run logger and set the remote type to `elasticsearch`: 
```
    ./logger generate --url=http://localhost:9200 --remote-type=elasticsearch
```

## Generate query requests to elasticsearch v6

```
    ./logger query --config ./config/dev.yaml --url http://localhost:9200 --remote-type=elasticsearch --logps 1
```
