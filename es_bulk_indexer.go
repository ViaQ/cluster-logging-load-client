package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/elastic/go-elasticsearch/v6/esutil"
)

const (
	indexName = "logger"
)

var (
	countSuccessful uint64

	numWorkers = runtime.NumCPU()
	flushBytes = 5e+6
	c          = make(chan os.Signal)
)

func init() {
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
}

type logContent struct {
	Hostname  string `json:"hostname"`
	Service   string `json:"service"`
	Level     string `json:"level"`
	Component string `json:"component"`
	Body      string `json:"body"`
}

// logViaEsCli uses bulkIndex to inject docs to es
func logViaEsCli(apiURL string, hostname string) {
	retryBackoff := backoff.NewExponentialBackOff()
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{apiURL},
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
		log.Fatalf("Error creating the client: %s", err)
	}

	// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//
	// Create the BulkIndexer
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         indexName,       // The default index name
		DocumentType:  "_doc",          // The default document type
		Client:        es,              // The Elasticsearch client
		NumWorkers:    numWorkers,      // The number of worker goroutines
		FlushBytes:    int(flushBytes), // The flush threshold in bytes
		FlushInterval: 2 * time.Second, // The periodic flush interval
	})
	if err != nil {
		log.Fatalf("Error creating the indexer: %s", err)
	}

	// Re-create the index
	//
	var res *esapi.Response
	if res, err = es.Indices.Delete([]string{indexName}, es.Indices.Delete.WithIgnoreUnavailable(true)); err != nil || res.IsError() {
		log.Fatalf("Cannot delete index: %s", err)
	}
	res.Body.Close()
	res, err = es.Indices.Create(indexName)
	if err != nil {
		log.Fatalf("Cannot create index: %s", err)
	}
	if res.IsError() {
		log.Fatalf("Cannot create index: %s", res)
	}
	res.Body.Close()

	ticker := time.NewTicker(time.Second / time.Duration(*logPerSec))
	for {
		select {
		case <-c:
			log.Println("\r- Ctrl+C pressed in Terminal")
			if err := bi.Close(context.Background()); err != nil {
				log.Fatalf("Unexpected error: %s", err)
			}
			ticker.Stop()
			return
		case <-ticker.C:
			log.Println("Sending log")
			a := logContent{
				Hostname:  hostname,
				Service:   string(randService()),
				Level:     string(randLevel()),
				Component: string(randComponent()),
				Body:      randomLog(),
			}
			// Prepare the data payload: encode log to JSON
			//
			data, err := json.Marshal(a)
			if err != nil {
				log.Fatalf("Cannot encode article %d: %s", a.Body, err)
			}

			// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
			//
			// Add an item to the BulkIndexer
			//
			err = bi.Add(
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
						log.Printf("Injected doc ID: %v", res.DocumentID)
						atomic.AddUint64(&countSuccessful, 1)
					},

					// OnFailure is called for each failed operation
					OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
						if err != nil {
							log.Printf("ERROR: %s", err)
						} else {
							log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
						}
					},
				},
			)
			if err != nil {
				log.Fatalf("Unexpected error: %s", err)
			}
		}
	}
}
