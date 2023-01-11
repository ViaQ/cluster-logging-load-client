package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/ViaQ/cluster-logging-load-client/internal"
	"github.com/ViaQ/cluster-logging-load-client/internal/generator"
	"github.com/ViaQ/cluster-logging-load-client/internal/querier"
	log "github.com/sirupsen/logrus"
)

var (
	opts = internal.Options{}
	logLevel string
)

func init() {
	flag.StringVar(&logLevel, "log-level", "error", "Overwrite to control the level of logs emitted. Allowed values: debug, info, warning, error")
	flag.StringVar(&opts.Command, "command", "generate", "Overwrite to control if logs are generated or queried. Allowed values: generate, query.")
	flag.StringVar(&opts.Destination, "destination", "stdout", "Overwrite to control where logs are queried or written to. Allowed values: loki, elasticsearch, stdout, file.")
	flag.StringVar(&opts.OutputFile, "file", "output.txt", "The name of the file to write logs to. Only available for \"File\" destinations.")
	flag.StringVar(&opts.ClientURL, "url", "", "URL of Promtail, LogCLI, or Elasticsearch client.")
	flag.BoolVar(&opts.DisableSecurityCheck, "disable-security-check", false, "Disable security check in HTTPS client.")
	flag.IntVar(&opts.LogsPerSecond, "logs-per-second", 1, "The rate to generate logs. This rate may not always be achievable.")
	flag.StringVar(&opts.LogType, "log-type", "simple", "Overwrite to control the type of logs generated. Allowed values: simple, application, synthetic.")
	flag.StringVar(&opts.LogFormat, "log-format", "default", "Overwrite to control the format of logs generated. Allowed values: default, crio (mimic CRIO output), csv, json")
	flag.StringVar(&opts.LabelType, "label-type", "none", "Overwrite to control what labels are included in Loki logs. Allowed values: none, client, client-host")
	flag.IntVar(&opts.SyntheticPayloadSize, "synthetic-payload-size", 100, "Overwrite to control size of synthetic log line.")
	flag.StringVar(&opts.Tenant, "tenant", "test", "Loki tenant ID for writing logs.")
	flag.IntVar(&opts.QueriesPerMinute, "queries-per-minute", 1, "The rate to generate queries. This rate may not always be achievable.")
	flag.StringVar(&opts.Query, "query", "", "Query to use to get logs from storage.")
	flag.StringVar(&opts.QueryRange, "query-range", "1s", "Duration of time period to query for logs (Loki only).")

	flag.Parse()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	ll, err := log.ParseLevel(logLevel)
	if err != nil {
		ll = log.ErrorLevel
	}
	
	log.SetLevel(ll)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if configJSON, err := json.MarshalIndent(opts, "", "\t"); err != nil {
		log.Infof("configuration:\n%s\n", configJSON)
	}

	switch opts.Command {
	case "generate":
		generatorOpts := generator.Options{
			Client:               generator.ClientType(opts.Destination),
			ClientURL:            opts.ClientURL,
			FileName:             opts.OutputFile,
			Tenant:               opts.Tenant,
			DisableSecurityCheck: opts.DisableSecurityCheck,
			LogsPerSecond:        opts.LogsPerSecond,
		}
		logGenerator, err := generator.NewLogGenerator(generatorOpts)
		if err != nil {
			panic(err)
		}
		logGenerator.GenerateLogs(
			generator.LogType(opts.LogType),
			generator.Format(opts.LogFormat),
			opts.SyntheticPayloadSize,
			generator.LabelSetOptions(opts.LabelType),
		)
	case "query":
		querierOpts := querier.Options{
			Client:               querier.ClientType(opts.Destination),
			ClientURL:            opts.ClientURL,
			Tenant:               opts.Tenant,
			DisableSecurityCheck: opts.DisableSecurityCheck,
			QueriesPerMinute:     opts.QueriesPerMinute,
			QueryRange:           opts.QueryRange,
		}
		logQuerier, err := querier.NewLogQuerier(querierOpts)
		if err != nil {
			panic(err)
		}
		logQuerier.QueryLogs(opts.Query)
	default:
		panic(fmt.Errorf("unknown command :%s", opts.Command))
	}
}
