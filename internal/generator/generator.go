package generator

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ViaQ/cluster-logging-load-client/internal/clients"

	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esutil"
	promtail "github.com/grafana/loki/v3/clients/pkg/promtail/client"
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

// Options describes the settings that can be modified for the log generator
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

	LogType              string
	LogFormat            string
	LabelType            string
	SyntheticPayloadSize int
	UseRandomHostname    bool

	// PayloadConfig is the path to a payload config file (required for LogType "config")
	PayloadConfig string
	// LoaderID is substituted for {{LOADER_ID}} in payload templates
	LoaderID string
	// StartSignalFile, when set, blocks emission until the file exists
	StartSignalFile string
	// Duration bounds the emission window. Zero means unlimited
	Duration time.Duration
	// SentCountFile, when set with Duration, receives the emitted line count
	SentCountFile string
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
	logCount                 prometheus.Counter
	opts                     Options
	payload                  *PayloadSource
}

func NewLogGenerator(opts Options, registry *prometheus.Registry) (*LogGenerator, error) {
	generator := LogGenerator{
		opts: opts,
		rate: opts.LogsPerSecond,
		logCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "log_generator_messages_produced_total",
			Help: "Total number of messages produced by the log generator",
		}),
	}
	registry.MustRegister(
		generator.logCount,
	)

	if LogType(opts.LogType) == ConfigLogType {
		if opts.PayloadConfig == "" {
			return nil, fmt.Errorf("--payload-config is required when --log-type is %q", ConfigLogType)
		}
		cfg, err := LoadPayloadConfig(opts.PayloadConfig)
		if err != nil {
			return nil, err
		}
		if generator.payload, err = NewPayloadSource(cfg, opts.LoaderID); err != nil {
			return nil, err
		}
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

func (g *LogGenerator) Start(ctx context.Context, wg *sync.WaitGroup, _ chan<- error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.generateLogs(ctx)
	}()
}

func (g *LogGenerator) generateLogs(ctx context.Context) {
	host, err := os.Hostname()
	if err != nil {
		log.Fatalf("error getting hostname: %s", err)
	}
	defer g.deferClose()

	logHostname := host
	if g.opts.UseRandomHostname {
		logHostname = fmt.Sprintf("%s.%032X", host, rand.Uint64())
	}

	if g.opts.StartSignalFile != "" {
		if err := WaitForFile(ctx, g.opts.StartSignalFile, startSignalPollInterval); err != nil {
			log.Debug("Shutting down log generator before start signal...")
			return
		}
	}

	var deadline time.Time
	if g.opts.Duration > 0 {
		deadline = time.Now().Add(g.opts.Duration)
	}

	lineCount := g.emit(ctx, host, logHostname, deadline)

	if g.opts.Duration > 0 {
		if g.opts.SentCountFile != "" {
			if err := WriteSentCount(g.opts.SentCountFile, lineCount); err != nil {
				log.Fatalf("error writing sent count file: %s", err)
			}
		}
		// Never exit: a restart would reset {{SEQ}} and falsify loss accounting.
		idle(ctx)
	}
}

// emit runs the paced emission loop until ctx is cancelled or, when deadline is
// non-zero, the deadline passes. It returns the number of lines emitted.
func (g *LogGenerator) emit(ctx context.Context, host, logHostname string, deadline time.Time) int64 {
	var lineCount int64 = 0
	expired := func() bool { return !deadline.IsZero() && !time.Now().Before(deadline) }

	for {
		if err := ctx.Err(); err != nil {
			log.Debug("Shutting down log generator...")
			break
		}
		if expired() {
			break
		}

		next := time.Now().UTC().Add(1 * time.Second)

		for i := 0; i < g.rate; i++ {
			if expired() {
				break
			}
			var logLine string
			if g.payload != nil {
				logLine = g.payload.Next()
			} else {
				var err error
				logLine, err = RandomLog(LogType(g.opts.LogType), g.opts.SyntheticPayloadSize)
				if err != nil {
					log.Fatalf("error creating log: %s", err)
				}
			}

			formattedLogLine, err := FormatLog(Format(g.opts.LogFormat), logHostname, lineCount, logLine)
			if err != nil {
				log.Fatalf("error formating log: %s", err)
			}

			err = g.writeToDestination(host, formattedLogLine, LabelSetOptions(g.opts.LabelType))
			if err != nil {
				log.Fatalf("error writing log: %s", err)
			}

			g.logCount.Inc()
			lineCount++
		}

		current := time.Now().UTC()
		wakeAt := next
		if !deadline.IsZero() && deadline.Before(wakeAt) {
			wakeAt = deadline
		}
		if current.Before(wakeAt) {
			time.Sleep(wakeAt.Sub(current))
		}
	}
	return lineCount
}

const startSignalPollInterval = 100 * time.Millisecond

// WaitForFile blocks until path exists, polling at the given interval. It
// returns the context error if ctx is cancelled first.
func WaitForFile(ctx context.Context, path string, poll time.Duration) error {
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(poll):
		}
	}
}

// WriteSentCount writes the emitted line count to path via temp file and rename.
func WriteSentCount(path string, count int64) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(fmt.Sprintf("%d\n", count)), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// idle blocks until ctx is cancelled.
func idle(ctx context.Context) {
	<-ctx.Done()
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
