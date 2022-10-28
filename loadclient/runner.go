package loadclient

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esutil"
	promtail "github.com/grafana/loki/clients/pkg/promtail/client"
	logcli "github.com/grafana/loki/pkg/logcli/client"
	log "github.com/sirupsen/logrus"
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
	lineCount                int64
}

type commandType string

var opt = Options{}

func ExecuteMultiThreaded(options Options) {
	opt = options
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

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
				// initialize the range
				q.initQueryRange()
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
	r.lineCount = 0

	for {
		next := time.Now().UTC().Add(1 * time.Second)

		for i := 0; i < opt.LogLinesPerSec; i++ {
			r.runnerAction(r.lineCount)
			r.lineCount++
		}

		current := time.Now().UTC()
		if current.Before(next) {
			time.Sleep(next.Sub(current))
		}
	}
}
