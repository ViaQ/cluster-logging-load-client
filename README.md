# Loki log generator

This project is a simple golang application to generate random logs.

This app runs in a single goroutine, if you want to generate more load, scale the app.

To build the app run `make build`.

`./logger` to start outputting log to stdout you'll need Promtail or an agent to send the logs.
`./logger --url=http://localhost:3100/api/prom/push` to send log to Loki directly

By default the app log 500 logs per seconds, this can be increased or decrease using the flag `--logps`. For example
`./logger --logps=1` will log one line per second.

To build a docker image use `make build-image`.

To run the image use: `docker run ctovena/logger:0.1 --url=http://localhost:3100/api/prom/push`
