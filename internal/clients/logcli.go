package clients

import (
	"net/url"
	"time"

	logcli "github.com/grafana/loki/pkg/logcli/client"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/config"
	log "github.com/sirupsen/logrus"
)

// NewLogCLIClient creates a new logCLI client
func NewLogCLIClient(clientURL, tenant string, disableSecurityCheck bool) (*logcli.DefaultClient, error) {
	URL, err := url.Parse(clientURL)
	if err != nil {
		return nil, err
	}

	client := logcli.DefaultClient{
		Address: URL.String(),
		OrgID:   tenant,
		TLSConfig: config.TLSConfig{
			InsecureSkipVerify: disableSecurityCheck,
		},
	}

	if !disableSecurityCheck {
		client.BearerTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		client.TLSConfig.CAFile = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
	}

	return &client, nil
}

// QueryLogsWithLogCLI executes a query range action with logCLI
func QueryLogsWithLogCLI(client *logcli.DefaultClient, query string, queryRange time.Duration) error {
	now := time.Now()
	res, err := client.QueryRange(query, 4000, now.Add(queryRange), now, logproto.FORWARD, 0, 0, false)

	log.Infof("logcli query complete. status: %s, %d results, took %f \n", res.Status, res.Data.Statistics.Ingester.TotalLinesSent, res.Data.Statistics.Summary.ExecTime)

	if err != nil {
		return err
	}
	return nil
}
