package loadclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/ViaQ/cluster-logging-load-client/loadclient/internal"
	"github.com/elastic/go-elasticsearch/v6/esutil"
	kitlog "github.com/go-kit/kit/log"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/loki/clients/pkg/promtail/api"
	promtail "github.com/grafana/loki/clients/pkg/promtail/client"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

type logGenerator struct {
	runner
	getLogLineFromSource func() (string, error)
	writeToDestination   func(string) error
	formatter            func(hash string, messageCount int64, payload string) string
}

func (g *logGenerator) destinationStdOut(logLine string) error {
	fmt.Printf("%s", logLine)
	return nil
}

func (g *logGenerator) destinationFile(logLine string) error {
	file := g.file
	_, _ = fmt.Fprintf(file, "%s", logLine)
	return nil
}

func (g *logGenerator) generateDestinationLoki(logLine string) error {
	labelSet := model.LabelSet{}

	switch opt.Loki.Labels {
	case "none":
		labelSet = model.LabelSet{
			"client": "promtail",
		}
	case "host":
		labelSet = model.LabelSet{
			"client":   "promtail",
			"hostname": model.LabelValue(g.hash),
		}
	case "random":
		labelSet = model.LabelSet{
			"client":    "promtail",
			"hostname":  model.LabelValue(g.hash),
			"service":   randService(),
			"level":     randLevel(),
			"component": randComponent(),
		}
	default:
		err := fmt.Errorf("unrecognized LokiLabels %s", opt.Loki.Labels)
		panic(err)
	}

	g.promtailClient.Chan() <- api.Entry{
		Labels: labelSet,
		Entry:  logproto.Entry{Timestamp: time.Now(), Line: logLine},
	}

	return nil
}

func (g *logGenerator) generateDestinationElasticSearch(logLine string) error {
	a := internal.LogContent{
		Hostname:  g.hash,
		Service:   string(randService()),
		Level:     string(randLevel()),
		Component: string(randComponent()),
		Body:      logLine,
		CreatedAt: time.Now().Round(time.Second).UTC(),
	}

	// Prepare the data payload: encode log to JSON
	data, err := json.Marshal(a)
	if err != nil {
		log.Fatalf("Cannot encode article %s: %s", a.Body, err)
	}

	// Add an item to the BulkIndexer
	err = g.elasticsearchBulkIndexer.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			// Action field configures the operation to perform (index, create, delete, update)
			Action: "index",

			// DocumentID is the (optional) document ID
			// DocumentID: strconv.Itoa(a.ID),

			// Body is an `io.Reader` with the payload
			Body: bytes.NewReader(data),

			// OnSuccess is called for each successful operation
			OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
				log.Infof("Injected doc ID: %v", res.DocumentID)
				atomic.AddInt64(&internal.CountSuccessful, 1)
			},

			// OnFailure is called for each failed operation
			OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
				if err != nil {
					log.Infof("ERROR: %s", err)
				} else {
					log.Infof("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
					atomic.AddInt64(&internal.CountFail, 1)
				}
			},
		},
	)
	if err != nil {
		log.Fatalf("Unexpected error: %s", err)
	}

	return nil
}

func sourceSimple() (string, error) {
	line := getSimpleLogLine()
	return line, nil
}

func sourceApplication() (string, error) {
	line := getApplicationLogLine()
	return line, nil
}

func sourceSynthetic() (string, error) {
	line := getSyntheticLogLine(opt.SyntheticPayloadSize)
	return line, nil
}

func formatDefault(hash string, messageCount int64, payload string) string {
	return fmt.Sprintf("goloader seq - %s - %010d - %s\n", hash, messageCount, payload)
}

func formatCrio(hash string, messageCount int64, payload string) string {
	now := time.Now().Format(time.RFC3339Nano)
	return fmt.Sprintf("%s stdout F goloader seq - %s - %010d - %s\n", now, hash, messageCount, payload)
}

func formatCSV(hash string, messageCount int64, payload string) string {
	now := time.Now().Format(time.RFC3339Nano)
	return fmt.Sprintf("ts=%s stream=%s host=%s lvl=%s count=%d msg=%s\n", now, randStream(), hash, randLevel(), messageCount, payload)
}

func formatJson(hash string, messageCount int64, payload string) string {
	now := time.Now().Format(time.RFC3339Nano)
	mymap := map[string]interface{}{
		"ts":     now,
		"stream": randStream(),
		"host":   hash,
		"lvl":    randLevel(),
		"count":  messageCount,
		"msg":    payload,
	}
	j, err := json.Marshal(mymap)
	if err != nil {
		log.Fatalf("Cannot marshal to json %s: %s", mymap, err)
	}
	return fmt.Sprintln(string(j))
}

func randStream() string {
	var stream string
	switch rand.Intn(2) {
	case 1:
		stream = "stderr"
	default:
		stream = "stdout"
	}
	return stream
}

func (g *logGenerator) initGenerateSource() {
	switch opt.Source {
	case "simple":
		g.getLogLineFromSource = sourceSimple
	case "application":
		g.getLogLineFromSource = sourceApplication
	case "synthetic":
		g.getLogLineFromSource = sourceSynthetic
	default:
		err := fmt.Errorf("unrecognized Source %s", opt.Source)
		panic(err)

	}
}

func (g *logGenerator) initGenerateDestination() func() {
	var err error
	switch opt.Destination {
	case "stdout":
		g.writeToDestination = g.destinationStdOut
	case "file":
		g.file, err = os.Create(opt.OutputFile)
		if err != nil {
			log.Fatalf("Unable to create out file %s: %v", opt.OutputFile, err)
		}
		g.writeToDestination = g.destinationFile
	case "loki":
		g.promtailClient, err = initPromtailClient(opt.DestinationAPIURL, opt.Loki.TenantID, opt.DisableSecurityCheck)
		if err != nil {
			log.Fatalf("Unable to initialize promtail client %v", err)
		}
		g.deferClose = func() {
			g.promtailClient.Stop()
		}
		g.writeToDestination = g.generateDestinationLoki
	case "elasticsearch":
		g.elasticsearchClient, err = internal.EsClient(opt.DestinationAPIURL)
		if err != nil {
			log.Fatalf("Unable to initialize elasticsearch client %v", err)
		}
		g.elasticsearchBulkIndexer = internal.CreateESBulkIndexer(g.elasticsearchClient)
		internal.RecreateESIndex(g.elasticsearchClient)
		g.deferClose = func() {
			waitCount := 0
			for {
				count := internal.CountSuccessful + internal.CountFail
				if count >= g.lineCount {
					break
				}
				waitCount++
				if waitCount > 60 {
					err = fmt.Errorf("Waited for 60 seconds and still there are  pending elasticsearch writes, PANIC ")
					panic(err)
				}
				time.Sleep(time.Duration(1 * float64(time.Second)))
			}
			_ = g.elasticsearchBulkIndexer.Close(context.Background())
		}
		g.writeToDestination = g.generateDestinationElasticSearch
	default:
		err = fmt.Errorf("unrecognized Destination %s", opt.Destination)
		panic(err)
	}

	return g.deferClose
}

func (g *logGenerator) initGenerateFormat() {
	switch opt.LogFormat {
	case "default":
		g.formatter = formatDefault
	case "crio":
		g.formatter = formatCrio
	case "csv":
		g.formatter = formatCSV
	case "json":
		g.formatter = formatJson
	default:
		err := fmt.Errorf("unrecognized formatter %s", opt.LogFormat)
		panic(err)
	}
}

func GenerateLog(options Options) {
	ExecuteMultiThreaded(options)
}

func initPromtailClient(apiURL, tenantID string, disableSecurityCheck bool) (promtail.Client, error) {
	URL, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}

	clientConfig := config.HTTPClientConfig{}

	if disableSecurityCheck {
		clientConfig.TLSConfig = config.TLSConfig{
			InsecureSkipVerify: disableSecurityCheck,
		}
	} else {
		clientConfig.Authorization = &config.Authorization{
			CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		}
		clientConfig.TLSConfig = config.TLSConfig{
			CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
		}
	}

	config := promtail.Config{
		Client:    clientConfig,
		BatchWait: 1 * time.Second,
		BatchSize: 1024 * 1024, // ~ 1 MB
		Timeout:   time.Second * 30,
		BackoffConfig: backoff.Config{
			MinBackoff: time.Second * 1,
			MaxBackoff: time.Second * 5,
			MaxRetries: 5,
		},
		URL:      flagext.URLValue{URL: URL},
		TenantID: tenantID,
	}
	metrics := promtail.NewMetrics(nil, nil)
	logger := kitlog.NewLogfmtLogger(os.Stdout)

	client, err := promtail.New(metrics, config, nil, logger)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (g *logGenerator) generatorAction(linesCount int64) {
	log.Debugf("generatorAction on line number: %d", linesCount)
	logLine, err := g.getLogLineFromSource()
	if err != nil {
		panic(err)
	}
	formattedLogLine := g.formatter(g.hash, linesCount, logLine)
	err = g.writeToDestination(formattedLogLine)
	if err != nil {
		panic(err)
	}
}
