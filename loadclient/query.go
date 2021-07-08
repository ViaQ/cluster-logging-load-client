package loadclient

import (
	"encoding/json"
	"github.com/ViaQ/cluster-logging-load-client/loadclient/internal"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strings"
	"time"
)

func QueryLog(queries []string,options Options) {
	opt = options
	if opt.DestinationAPIURL != "" {
		switch opt.Destination {
		case "loki":
			log.Errorf("Query to loki: TO BE IMPLEMENTED")
		case "elasticsearch":
			log.Debugf("Query es")
			logQueryES(opt.DestinationAPIURL, queries, opt.LogLinesPerSec)
		default:
			log.Errorf("Unsupported remote type: %s\n", opt.Destination)
		}
		return
	}
}

func logQueryES(apiURL string, queries []string, logLinesPerSec int ) {
	log.Infof("%d queries: %v\n", len(queries), queries)
	es, err := internal.EsClient(apiURL)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	ticker := time.NewTicker(time.Second / time.Duration(logLinesPerSec))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			var b strings.Builder
			q := queries[rand.Intn(len(queries))]
			log.Infof("query: %v\n", q)
			b.WriteString(q)
			res, err := es.Search(
				es.Search.WithIndex(internal.IndexName),
				es.Search.WithBody(strings.NewReader(b.String())),
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

			log.Infof("took %d, hit total: %v\n", r.Took, r.Hits.Total)
			_ = res.Body.Close()
		}
	}
}



