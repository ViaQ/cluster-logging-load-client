package loadclient

import (
	"encoding/json"
	"fmt"
	"github.com/ViaQ/cluster-logging-load-client/loadclient/internal"
	logcli "github.com/grafana/loki/pkg/logcli/client"
	"github.com/grafana/loki/pkg/logproto"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/rand"
	"net/url"
	"strings"
	"time"
)

type logQuerier struct {
	runner
	queryFrom func(query string, lineCount int64) error
	queries   []string
}

func (q *logQuerier) initQueryDestination() {
	var err error
	switch opt.Destination {
	case "loki":
		q.lokiLogCLIClient, err = initLogCLIClient(opt.DestinationAPIURL, opt.Loki.TenantID)
		if err != nil {
			log.Fatalf("Unable to initialize logcli client %v", err)
		}
		q.queryFrom = q.queryLoki
	case "elasticsearch":
		q.elasticsearchClient, err = internal.EsClient(opt.DestinationAPIURL)
		if err != nil {
			log.Fatalf("Error creating the client: %s", err)
		}
		q.queryFrom = q.queryElasticSearch
	default:
		err = fmt.Errorf("unrecognized Destination %s", opt.Destination)
		panic(err)
	}
}

func (q *logQuerier) queryLoki(query string, count int64) error {
	log.Infof("query: %v\n", query)

	resp, err := q.lokiLogCLIClient.QueryRange(query, 1000, time.Unix(0, 0), time.Now(), logproto.FORWARD, 0, 0, false)
	if err != nil {
		log.Fatalf("Error Query using  loki logcli: %s", err)
	}

	log.Infof("query count %d :: status is %s, %d results, took %f \n", count, resp.Status, resp.Data.Statistics.Ingester.TotalLinesSent, resp.Data.Statistics.Summary.ExecTime)
	return nil
}

func (q *logQuerier) queryElasticSearch(query string, count int64) error {
	var b strings.Builder
	log.Infof("query: %v\n", query)
	b.WriteString(query)
	res, err := q.elasticsearchClient.Search(
		q.elasticsearchClient.Search.WithIndex(internal.IndexName),
		q.elasticsearchClient.Search.WithBody(strings.NewReader(b.String())),
	)
	if err != nil {
		log.Fatalf("Error getting search response: %s", err)
	}

	type envelopeResponse struct {
		Took int
		Hits struct {
			Total int
			Hits  []struct {
				ID         string          `json:"_id"`
				Source     json.RawMessage `json:"_source"`
				Highlights json.RawMessage `json:"highlight"`
				Sort       []interface{}   `json:"sort"`
			}
		}
	}

	var r envelopeResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Fatalf("Error parsing search response: %s", err)
	}

	log.Infof("query count %d :: status is %s, %d results, took %f \n", count, "success", r.Hits.Total, float64(r.Took)/1000)
	_ = res.Body.Close()
	return nil
}

func initLogCLIClient(apiURL string, tenantID string) (logcli.DefaultClient, error) {
	URL, err := url.Parse(apiURL)
	if err != nil {
		panic(err)
	}
	logCLIClient := logcli.DefaultClient{
		Address: URL.String(),
		OrgID:   tenantID,
	}
	return logCLIClient, nil
}

type queryYamlFormat struct {
	Query []string `yaml:"query"`
}

func QueryLog(options Options) {
	ExecuteMultiThreaded(options)
}

func (q *logQuerier) initQueries() {
	if opt.QueryFile != "" {
		yamlFile, err := ioutil.ReadFile(opt.QueryFile)
		if err != nil {
			log.Fatalf("can't open query yaml file %s [%v]", opt.QueryFile, err)
		}
		err = yaml.Unmarshal(yamlFile, &q.queries)
		if err != nil {
			log.Fatalf("can't unmarshal query yaml file %s [%v]", opt.QueryFile, err)
		}
	} else if len(opt.Queries)>0 {
		q.queries = opt.Queries
	} else {
		panic("can't find queries to use. Not using file and not using command line parameters")
	}

	log.Infof("%d queries: %v\n", len(q.queries), q.queries)
}

func (q *logQuerier) getQuery() string {
	query := q.queries[rand.Intn(len(q.queries))]
	return query
}

func (q *logQuerier) queryAction(linesCount int64) {
	query := q.getQuery()

	err := q.queryFrom(query, linesCount)
	if err != nil {
		panic(err)
	}
}