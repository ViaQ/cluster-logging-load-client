package internal

import (
	"fmt"
	"github.com/elastic/go-elasticsearch/v6"
	log "github.com/sirupsen/logrus"
	"runtime"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/elastic/go-elasticsearch/v6/esutil"
)

const (
	IndexName = "logger"
)

var (
	CountSuccessful int64
	CountFail       int64

	numWorkers = runtime.NumCPU()
	flushBytes = 5e+6
)

type LogContent struct {
	Hostname  string    `json:"hostname"`
	Service   string    `json:"service"`
	Level     string    `json:"level"`
	Component string    `json:"component"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func EsClient(esURL string) (*elasticsearch.Client, error) {
	retryBackoff := backoff.NewExponentialBackOff()
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{esURL},
		// Retry on 429 TooManyRequests statuses
		//
		RetryOnStatus: []int{502, 503, 504, 429},
		RetryBackoff: func(i int) time.Duration {
			if i == 1 {
				retryBackoff.Reset()
			}
			return retryBackoff.NextBackOff()
		},
		MaxRetries: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating the client: %s", err)
	}
	return es, nil
}

func CreateESBulkIndexer(es *elasticsearch.Client) esutil.BulkIndexer {
	// Create the BulkIndexer
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         IndexName,       // The default index name
		DocumentType:  "_doc",          // The default document type
		Client:        es,              // The Elasticsearch client
		NumWorkers:    numWorkers,      // The number of worker goroutines
		FlushBytes:    int(flushBytes), // The flush threshold in bytes
		FlushInterval: 2 * time.Second, // The periodic flush interval
	})
	if err != nil {
		log.Fatalf("Error creating the indexer: %s", err)
	}

	return bi
}

func RecreateESIndex(es *elasticsearch.Client) {
	var err error
	// Re-create the index
	var res *esapi.Response
	if res, err = es.Indices.Delete([]string{IndexName}, es.Indices.Delete.WithIgnoreUnavailable(true)); err != nil || res.IsError() {
		log.Fatalf("Cannot delete index: %s", err)
	}
	_ = res.Body.Close()
	res, err = es.Indices.Create(IndexName)
	if err != nil {
		log.Fatalf("Cannot create index: %s", err)
	}
	if res.IsError() {
		log.Fatalf("Cannot create index: %s", res)
	}
	_ = res.Body.Close()
}
