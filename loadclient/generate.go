package loadclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ViaQ/cluster-logging-load-client/loadclient/internal"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esutil"
	kitlog "github.com/go-kit/kit/log"
	promtail "github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type logGenerator struct {
	file                     io.Writer
	hash                     string
	getLogLineFromSource     func() (string, error)
	writeToDestination       func(string) error
	deferClose               func()
	formatter                func(hash string, messageCount int, payload string) string
	promtailClient           promtail.Client
	elasticsearchClient      *elasticsearch.Client
	elasticsearchBulkIndexer esutil.BulkIndexer
}

var opt = Options{}

const (
	minBurstMessageCount = 100
	numberOfBursts       = 10
)

func (g *logGenerator) destinationStdOut(logLine string) error {
	fmt.Printf("%s", logLine)
	return nil
}

func (g *logGenerator) destinationFile(logLine string) error {
	file := g.file
	_, _ = fmt.Fprintf(file, "%s", logLine)
	return nil
}

func (g *logGenerator) destinationLoki(logLine string) error {
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
	err := g.promtailClient.Handle(labelSet, time.Now(), logLine)
	if err != nil {
		log.Errorf("destinationLoki error  %s", err)
	}

	return nil
}

func (g *logGenerator) destinationElasticSearch(logLine string) error {
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
				atomic.AddUint64(&internal.CountSuccessful, 1)
			},

			// OnFailure is called for each failed operation
			OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
				if err != nil {
					log.Infof("ERROR: %s", err)
				} else {
					log.Infof("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
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

func formatDefault(hash string, messageCount int, payload string) string {
	return fmt.Sprintf("goloader seq - %s - %010d - %s\n", hash, messageCount, payload)
}

func formatCrio(hash string, messageCount int, payload string) string {
	now := time.Now().Format(time.RFC3339Nano)
	return fmt.Sprintf("%s stdout F goloader seq - %s - %010d - %s\n", now, hash, messageCount, payload)
}

func formatCSV(hash string, messageCount int, payload string) string {
	now := time.Now().Format(time.RFC3339Nano)
	return fmt.Sprintf("ts=%s stream=%s host=%s lvl=%s count=%d msg=%s\n", now, randStream(), hash, randLevel(), messageCount, payload)
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

func (g *logGenerator) initSource() {
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

func (g *logGenerator) initDestination() func() {
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
		g.promtailClient, err = initPromtailClient(opt.DestinationAPIURL, opt.Loki.TenantID)
		if err != nil {
			log.Fatalf("Unable to initialize promtail client %v", err)
		}
		g.deferClose = func() {
			g.promtailClient.Stop()
		}
		g.writeToDestination = g.destinationLoki
	case "elasticsearch":
		g.elasticsearchClient, err = internal.EsClient(opt.DestinationAPIURL)
		if err != nil {
			log.Fatalf("Unable to initialize elasticsearch client %v", err)
		}
		g.elasticsearchBulkIndexer = internal.CreateESBulkIndexer(g.elasticsearchClient)
		internal.RecreateESIndex(g.elasticsearchClient)
		g.deferClose = func() {
			_ = g.elasticsearchBulkIndexer.Close(context.Background())
		}
		g.writeToDestination = g.destinationElasticSearch
	default:
		err = fmt.Errorf("unrecognized Destination %s", opt.Destination)
		panic(err)
	}

	return g.deferClose
}

func (g *logGenerator) initFormat() {
	switch opt.LogFormat {
	case "default":
		g.formatter = formatDefault
	case "crio":
		g.formatter = formatCrio
	case "csv":
		g.formatter = formatCSV
	default:
		err := fmt.Errorf("unrecognized formatter %s", opt.LogFormat)
		panic(err)
	}
}

func (g *logGenerator) run() {
	burstSize := 1
	if opt.LogLinesPerSec > minBurstMessageCount {
		burstSize = numberOfBursts
	}

	linesCount := 0
	startTime := time.Now().Unix() - 1
	sleep := 1.0 / float64(burstSize)

	for {
		for i := 0; i < opt.LogLinesPerSec/burstSize; i++ {
			logLine, err := g.getLogLineFromSource()
			if err != nil {
				panic(err)
			}
			formattedLogLine := g.formatter(g.hash, linesCount, logLine)
			err = g.writeToDestination(formattedLogLine)
			if err != nil {
				panic(err)
			}
			linesCount++
			if opt.TotalLogLines != 0 && linesCount >= opt.TotalLogLines {
				return
			}
		}
		deltaTime := int(time.Now().Unix() - startTime)

		messagesLoggedPerSec := linesCount / deltaTime
		if messagesLoggedPerSec >= opt.LogLinesPerSec {
			time.Sleep(time.Duration(sleep * float64(time.Second)))
		}
	}
}

func GenerateLog(options Options) {
	opt = options
	var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	var runnersWaitGroup sync.WaitGroup
	runnersWaitGroup.Add(opt.Threads)
	for threadCount := 0; threadCount < opt.Threads; threadCount++ {
		go func(threadCount int) {
			g := logGenerator{}
			// define hash
			g.hash = fmt.Sprintf("%s.%d.%032X", host, threadCount, rnd.Uint64())
			// define Source for log lines
			g.initSource()
			// define Destination for log lines
			deferFunc := g.initDestination()
			if deferFunc != nil {
				defer deferFunc()
			}
			// define log line format
			g.initFormat()
			// run
			log.Infof("Start generating logs on thread #%d", threadCount)
			g.run()
			runnersWaitGroup.Done()
			log.Infof("Done generating logs on thread #%d", threadCount)
		}(threadCount)
	}
	runnersWaitGroup.Wait()
}

func initPromtailClient(apiURL string, tenantID string) (promtail.Client, error) {
	URL, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	promtailClient, err := promtail.New(promtail.Config{
		BatchWait: 0,
		BatchSize: 1000,
		Timeout:   time.Second * 30,
		BackoffConfig: util.BackoffConfig{
			MinBackoff: time.Second * 1,
			MaxBackoff: time.Second * 5,
			MaxRetries: 5,
		},
		URL:      flagext.URLValue{URL: URL},
		TenantID: tenantID,
	}, logger)
	if err != nil {
		return nil, err
	}
	return promtailClient, nil
}
