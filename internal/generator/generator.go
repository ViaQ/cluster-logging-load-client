package loadclient

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ViaQ/cluster-logging-load-client/internal/clients"

	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
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
	elasticsearchClient *elasticsearch.Client
	elasticsearchBulkIndexer esutil.BulkIndexer
	file *File
	promtailClient           *promtail.Client
	host string
	rate                int
	writeToDestination func(string) error
	deferClose               func()
}

func NewLogGenerator(opts Options) (*LogGenerator, error) {
	generator := LogGenerator{
		rate: opts.LogsPerSecond,
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("error getting hostname: %s", err)
	}
	generator.host = host

	switch opt.Destination {
	case "file":
		outFile, err := os.Create(opts.FileName)
		if err != nil {
			return nil, fmt.Errorf("Unable to create out file %s: %v", opt.OutputFile, err)
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

func (g *logGenerator) GenerateLog(logType LogType, logFormat Format) {
	lineCount = 0

	defer g.deferClose()

	for {
		next := time.Now().UTC().Add(1 * time.Second)

		for i := 0; i < opt.LogLinesPerSec; i++ {
			logLine, err := RandomLog(logType, opt.SyntheticPayloadSize)
			if err != nil {
				log.Fatalf("error creating log: %s", err) 
			}

			formattedLogLine, err := FormatLog(logFormat, generator.host, lineCount, logLine)
			if err != nil {
				log.Fatalf("error formating log: %s", err) 
			}

			err = g.writeToDestination(formattedLogLine)
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

func (g *logGenerator) writeLogToStdout(logLine string) error {
	fmt.Printf("%s", logLine)
	return nil
}

func (g *logGenerator) writeLogToFile(logLine string) error {
	_, err = fmt.Fprintf(g.file, "%s", logLine)
	if err != nil {
		return err
	}
	return nil
}

func (g *logGenerator) sendLokiLog(logLine string) error {
	labels := LogLabelSet(g.host, LabelSetOptions(opt.Loki.Labels))
	clients.SendLogWithPromtail(g.promtailClient, logLine, labels)
	return nil
}

func (g *logGenerator) sendElasticsearchLog(logLine string) error {
	content, err := ElasticsearchLogContent(g.host)
	if err != nil {
		return err
	}
	return clients.SendLogWithElasticsearch(g.elasticsearchBulkIndexer, content)
}
