package generator

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/ViaQ/cluster-logging-load-client/internal/clients"

	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esutil"
	promtail "github.com/grafana/loki/clients/pkg/promtail/client"
	log "github.com/sirupsen/logrus"
)

// ClientType describes the type of client to use for querying logs
type ClientType string

const (
	// FileClientType uses a file to write logs to
	FileClientType ClientType = "file"

	// LokiClientType uses a Promtail client to forward logs
	LokiClientType ClientType = "loki"

	// ElasticsearchClientType uses an Elasticsearch client to forward logs
	ElasticsearchClientType ClientType = "elasticsearch"
)

// Options describes the settings that can be modified for the querier
type Options struct {
	// Client describes the client to use for forwarding
	Client ClientType
	// ClientURl is the endpoint to forward to
	ClientURL string
	// FileName is the name of the file to create and write to
	FileName string
	// Tenant is identification to use for Loki
	Tenant string
	// DisableSecurityCheck deactivates the TLS checks
	DisableSecurityCheck bool
	// LogsPerSecond is the number of logs to write per second
	LogsPerSecond int
}

// LogGenerator describes an object which generates logs
type LogGenerator struct {
	elasticsearchClient      *elasticsearch.Client
	elasticsearchBulkIndexer esutil.BulkIndexer
	file                     *os.File
	promtailClient           promtail.Client
	rate                     int
	writeToDestination       func(string, string, LabelSetOptions) error
	deferClose               func()
}

func NewLogGenerator(opts Options) (*LogGenerator, error) {
	generator := LogGenerator{
		rate: opts.LogsPerSecond,
	}

	switch opts.Client {
	case "file":
		outFile, err := os.Create(opts.FileName)
		if err != nil {
			return nil, fmt.Errorf("Unable to create out file %s: %v", opts.FileName, err)
		}

		generator.file = outFile
		generator.writeToDestination = generator.writeLogToFile
		generator.deferClose = func() {
			fmt.Println("done")
		}
	case "loki":
		client, err := clients.NewPromtailClient(opts.ClientURL, opts.Tenant, opts.DisableSecurityCheck)
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize promtail client %v", err)
		}

		generator.promtailClient = client
		generator.writeToDestination = generator.sendLokiLog
		generator.deferClose = func() {
			generator.promtailClient.Stop()
		}
	case "elasticsearch":
		client, err := clients.NewElasticsearchClient(opts.ClientURL)
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize elasticsearch client %v", err)
		}
		indexer, err := clients.NewElasticsearchBulkIndexer(client)
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize elasticsearch client %v", err)
		}
		if err = clients.RecreateElasticsearchIndex(client, clients.IndexName); err != nil {
			return nil, fmt.Errorf("Unable to initialize elasticsearch client %v", err)
		}

		generator.elasticsearchClient = client
		generator.elasticsearchBulkIndexer = indexer
		generator.writeToDestination = generator.sendElasticsearchLog
		generator.deferClose = func() {
			_ = generator.elasticsearchBulkIndexer.Close(context.Background())
		}
	default:
		generator.writeToDestination = generator.writeLogToStdout
		generator.deferClose = func() {
			fmt.Println("done")
		}
	}

	return &generator, nil
}

func (g *LogGenerator) GenerateLogs(logType LogType, logFormat Format, logSize int, labelOpts LabelSetOptions, randomizeHostname bool) {
	host, err := os.Hostname()
	if err != nil {
		log.Fatalf("error getting hostname: %s", err)
	}

	defer g.deferClose()

	var lineCount int64 = 0

	logHostname := host
	if randomizeHostname {
		logHostname = fmt.Sprintf("%s.%032X", host, rand.Uint64())
	}

	for {
		next := time.Now().UTC().Add(1 * time.Second)

		for i := 0; i < g.rate; i++ {
			logLine, err := RandomLog(logType, logSize)
			if err != nil {
				log.Fatalf("error creating log: %s", err)
			}

			formattedLogLine, err := FormatLog(logFormat, logHostname, lineCount, logLine)
			if err != nil {
				log.Fatalf("error formating log: %s", err)
			}

			err = g.writeToDestination(host, formattedLogLine, labelOpts)
			if err != nil {
				log.Fatalf("error writing log: %s", err)
			}

			lineCount++
		}

		current := time.Now().UTC()
		if current.Before(next) {
			time.Sleep(next.Sub(current))
		}
	}
}

func (g *LogGenerator) writeLogToStdout(host, logLine string, labelOpts LabelSetOptions) error {
	fmt.Printf("%s", logLine)
	return nil
}

func (g *LogGenerator) writeLogToFile(host, logLine string, labelOpts LabelSetOptions) error {
	_, err := fmt.Fprintf(g.file, "%s", logLine)
	if err != nil {
		return err
	}
	return nil
}

func (g *LogGenerator) sendLokiLog(host, logLine string, labelOpts LabelSetOptions) error {
	labels := LogLabelSet(host, LabelSetOptions(labelOpts))
	clients.SendLogWithPromtail(g.promtailClient, logLine, labels)
	return nil
}

func (g *LogGenerator) sendElasticsearchLog(host, logLine string, labelOpts LabelSetOptions) error {
	content, err := NewElasticsearchLogContent(host, logLine)
	if err != nil {
		return err
	}
	return clients.SendLogWithElasticsearch(g.elasticsearchBulkIndexer, content)
}
