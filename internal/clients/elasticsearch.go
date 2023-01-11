package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esutil"
	log "github.com/sirupsen/logrus"
)

const (
	IndexName = "logger"
)

func NewElasticsearchClient(clientURL string) (*elasticsearch.Client, error) {
	retryBackoff := backoff.NewExponentialBackOff()
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses:     []string{clientURL},
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
	return client, nil
}

func NewElasticsearchBulkIndexer(client *elasticsearch.Client) (esutil.BulkIndexer, error) {
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         IndexName,        // The default index name
		DocumentType:  "_doc",           // The default document type
		Client:        client,           // The Elasticsearch client
		NumWorkers:    runtime.NumCPU(), // The number of worker goroutines
		FlushBytes:    int(5e+6),        // The flush threshold in bytes
		FlushInterval: 2 * time.Second,  // The periodic flush interval
	})
	if err != nil {
		return nil, fmt.Errorf("error creating the indexer: %s", err)
	}
	return bi, nil
}

func RecreateElasticsearchIndex(client *elasticsearch.Client, index string) error {
	if err := deleteIndex(client, index); err != nil {
		return err
	}
	if err := createIndex(client, index); err != nil {
		return err
	}
	return nil
}

func SendLogWithElasticsearch(indexer esutil.BulkIndexer, logData []byte) error {
	// Add an item to the BulkIndexer
	err := indexer.Add(
		context.Background(),
		esutil.BulkIndexerItem{
			// Action field configures the operation to perform (index, create, delete, update)
			Action: "index",

			// DocumentID is the (optional) document ID
			// DocumentID: strconv.Itoa(a.ID),

			// Body is an `io.Reader` with the payload
			Body: bytes.NewReader(logData),

			// OnSuccess is called for each successful operation
			OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
				log.Infof("Injected doc ID: %v", res.DocumentID)
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
		return fmt.Errorf("error sending log with bulk indexer: %s", err)
	}
	return nil
}

func QueryLogsWithElasticsearch(client *elasticsearch.Client, index, query string) error {
	res, err := client.Search(
		client.Search.WithIndex(index),
		client.Search.WithBody(strings.NewReader(query)),
	)
	defer res.Body.Close()

	if err != nil {
		return fmt.Errorf("error getting search response: %s", err)
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
		return fmt.Errorf("error parsing search response: %s", err)
	}

	log.Infof("elasticsearch query complete. status is %s, %d results, took %f \n", "success", r.Hits.Total, float64(r.Took)/1000)
	return nil
}

func createIndex(client *elasticsearch.Client, index string) error {
	res, err := client.Indices.Create(index)
	defer res.Body.Close()

	if err != nil || res.IsError() {
		return fmt.Errorf("error creating index %s: %s", index, err)
	}
	return nil
}

func deleteIndex(client *elasticsearch.Client, index string) error {
	res, err := client.Indices.Delete(
		[]string{index},
		client.Indices.Delete.WithIgnoreUnavailable(true),
	)
	defer res.Body.Close()

	if err != nil || res.IsError() {
		return fmt.Errorf("error deleting index %s: %s", index, err)
	}
	return nil
}
