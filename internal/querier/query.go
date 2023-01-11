package querier

import (
	"fmt"
	"time"

	"github.com/ViaQ/cluster-logging-load-client/internal/clients"

	"github.com/elastic/go-elasticsearch/v6"
	logcli "github.com/grafana/loki/pkg/logcli/client"
	log "github.com/sirupsen/logrus"
)

// ClientType describes the type of client to use for querying logs
type ClientType string

const (
	// LokiClientType uses a LogCLI client to execute query range actions
	LokiClientType ClientType = "loki"

	// ElasticsearchClientType uses an Elasticsearch client to execute query actions
	ElasticsearchClientType ClientType = "elasticsearch"
)

// Options describes the settings that can be modified for the querier
type Options struct {
	// Client describes the client to use for querying
	Client ClientType
	// ClientURl is the endpoint to query against
	ClientURL string
	// Tenant is identification to use for Loki
	Tenant string
	// DisableSecurityCheck deactivates the TLS checks
	DisableSecurityCheck bool
	// QueriesPerMinute is the number of queries to launch per minute
	QueriesPerMinute int
	// QueryRange is the range over which LogCLI will query against
	QueryRange string
}

// LogQuerier describes an object which queries for logs
type LogQuerier struct {
	elasticsearchClient *elasticsearch.Client
	logCLIClient        *logcli.DefaultClient
	rate                int
	queryFrom           func(string) error
	queryRange          time.Duration
}

// NewLogQuerier creates a new querier object
func NewLogQuerier(opts Options) (*LogQuerier, error) {
	querier := LogQuerier{
		rate: opts.QueriesPerMinute,
	}

	switch opts.Client {
	case ElasticsearchClientType:
		client, err := clients.NewElasticsearchClient(opts.ClientURL)
		if err != nil {
			return nil, err
		}

		querier.elasticsearchClient = client
		querier.queryFrom = querier.queryElasticSearch
	case LokiClientType:
		client, err := clients.NewLogCLIClient(opts.ClientURL, opts.Tenant, opts.DisableSecurityCheck)
		if err != nil {
			return nil, err
		}

		rangeDuration, err := time.ParseDuration(opts.QueryRange)
		if err != nil {
			return nil, err
		}

		querier.logCLIClient = client
		querier.queryFrom = querier.queryLoki
		querier.queryRange = rangeDuration
	default:
		return nil, fmt.Errorf("error client type: %s", opts.Client)
	}

	return &querier, nil
}

// QueryLogs indefinitely queries logs using the configured client
func (q *LogQuerier) QueryLogs(query string) {
	for {
		next := time.Now().UTC().Add(1 * time.Minute)

		for i := 0; i < q.rate; i++ {
			if err := q.queryFrom(query); err != nil {
				log.Fatalf("error querying: %s", err)
			}
		}

		current := time.Now().UTC()
		if current.Before(next) {
			time.Sleep(next.Sub(current))
		}
	}
}

func (q *LogQuerier) queryLoki(query string) error {
	return clients.QueryLogsWithLogCLI(q.logCLIClient, query, q.queryRange)
}

func (q *LogQuerier) queryElasticSearch(query string) error {
	return clients.QueryLogsWithElasticsearch(q.elasticsearchClient, clients.IndexName, query)
}
