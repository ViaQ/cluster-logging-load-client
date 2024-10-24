package main

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/ViaQ/cluster-logging-load-client/internal"
	"github.com/ViaQ/cluster-logging-load-client/internal/generator"
	"github.com/ViaQ/cluster-logging-load-client/internal/querier"
)

var (
	opts     = internal.Options{}
	logLevel string
)

func init() {
	pflag.StringVar(&logLevel, "log-level", "error", "Overwrite to control the level of logs emitted. Allowed values: debug, info, warning, error")
	pflag.StringVar(&opts.Command, "command", "generate", "Overwrite to control if logs are generated or queried. Allowed values: generate, query.")
	pflag.StringVar(&opts.Destination, "destination", "stdout", "Overwrite to control where logs are queried or written to. Allowed values: loki, elasticsearch, stdout, file.")
	pflag.StringVar(&opts.OutputFile, "file", "output.txt", "The name of the file to write logs to. Only available for \"File\" destinations.")
	pflag.StringVar(&opts.ClientURL, "url", "", "URL of Promtail, LogCLI, or Elasticsearch client.")
	pflag.BoolVar(&opts.DisableSecurityCheck, "disable-security-check", false, "Disable security check in HTTPS client.")
	pflag.IntVar(&opts.LogsPerSecond, "logs-per-second", 1, "The rate to generate logs. This rate may not always be achievable.")
	pflag.StringVar(&opts.LogType, "log-type", "simple", "Overwrite to control the type of logs generated. Allowed values: application, audit, simple, synthetic.")
	pflag.StringVar(&opts.LogFormat, "log-format", "default", "Overwrite to control the format of logs generated. Allowed values: default, crio (mimic CRIO output), csv, json, raw")
	pflag.StringVar(&opts.LabelType, "label-type", "none", "Overwrite to control what labels are included in Loki logs. Allowed values: none, client, client-host")
	pflag.BoolVar(&opts.UseRandomHostname, "use-random-hostname", false, "Ensures that the hostname field is unique by adding a random integer to the end.")
	pflag.IntVar(&opts.SyntheticPayloadSize, "synthetic-payload-size", 100, "Overwrite to control size of synthetic log line.")
	pflag.StringVar(&opts.Tenant, "tenant", "test", "Loki tenant ID for writing logs.")
	pflag.IntVar(&opts.QueriesPerMinute, "queries-per-minute", 1, "The rate to generate queries. This rate may not always be achievable.")
	pflag.StringVar(&opts.Query, "query", "", "Query to use to get logs from storage.")
	pflag.StringVar(&opts.QueryRange, "query-range", "1s", "Duration of time period to query for logs (Loki only).")

	pflag.Parse()
}

func main() {
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
			opts.UseRandomHostname,
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
