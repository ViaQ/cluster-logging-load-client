package loadclient

import (
	"fmt"
	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esutil"
	logcli "github.com/grafana/loki/pkg/logcli/client"
	promtail "github.com/grafana/loki/pkg/promtail/client"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	Generate = "generate"
	Query    = "query"
)

type runner struct {
	file                     io.Writer
	hash                     string
	runnerAction             func(int64)
	deferClose               func()
	promtailClient           promtail.Client
	lokiLogCLIClient         logcli.DefaultClient
	elasticsearchClient      *elasticsearch.Client
	elasticsearchBulkIndexer esutil.BulkIndexer
	lineCount				 int64
}

type commandType string

var opt = Options{}

func ExecuteMultiThreaded(options Options) {
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
			switch opt.Command {
			case Generate:
				g := logGenerator{}
				// define hash
				g.hash = fmt.Sprintf("%s.%d.%032X", host, threadCount, rnd.Uint64())
				// define Source for log lines
				g.initGenerateSource()
				// define Destination for log lines
				g.initGenerateDestination()
				// define log line format
				g.initGenerateFormat()
				// define the runner action
				g.runnerAction = g.generatorAction
				// run
				log.Infof("Start running on thread #%d", threadCount)
				g.run()
				if g.deferClose != nil {
					g.deferClose()
				}
				log.Infof("Done running on thread #%d", threadCount)
				runnersWaitGroup.Done()
			case Query:
				q := logQuerier{}
				// initialize the list of queries
				q.initQueries()
				// define Destination for queries
				q.initQueryDestination()
				// define the runner action
				q.runnerAction = q.queryAction
				// run
				log.Infof("Start running on thread #%d", threadCount)
				q.run()
				if q.deferClose != nil {
					q.deferClose()
				}
				log.Infof("Done running on thread #%d", threadCount)
				runnersWaitGroup.Done()
			default:
				err = fmt.Errorf("unrecognized Command %s", opt.Command)
				panic(err)
			}
		}(threadCount)
	}
	runnersWaitGroup.Wait()
}

func (r *runner) run() {
	burstSize := int64(1)
	if opt.LogLinesPerSec > minBurstMessageCount {
		burstSize = numberOfBursts
	}

	r.lineCount = 0
	startTime := time.Now().Unix() - 1
	sleep := 1.0 / float64(burstSize)

	for {
		for i := int64(0); i < opt.LogLinesPerSec/burstSize; i++ {
			// execute the per-log-line action
			r.runnerAction(r.lineCount)
			r.lineCount++
			if opt.TotalLogLines != 0 && r.lineCount >= opt.TotalLogLines {
				return
			}
		}
		deltaTime := int64(time.Now().Unix() - startTime)

		messagesLoggedPerSec := r.lineCount / deltaTime
		if messagesLoggedPerSec >= opt.LogLinesPerSec {
			time.Sleep(time.Duration(sleep * float64(time.Second)))
		}
	}
}
