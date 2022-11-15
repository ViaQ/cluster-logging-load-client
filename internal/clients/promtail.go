package clients

import (
	"net/url"
	"os"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/loki/clients/pkg/promtail/api"
	promtail "github.com/grafana/loki/clients/pkg/promtail/client"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

// NewPromtailClient creates a Promtail client
func NewPromtailClient(clientURL, tenantID string, disableSecurityCheck bool) (promtail.Client, error) {
	URL, err := url.Parse(clientURL)
	if err != nil {
		return nil, err
	}

	clientConfig := config.HTTPClientConfig{}

	if disableSecurityCheck {
		clientConfig.TLSConfig = config.TLSConfig{
			InsecureSkipVerify: disableSecurityCheck,
		}
	} else {
		clientConfig.Authorization = &config.Authorization{
			CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		}
		clientConfig.TLSConfig = config.TLSConfig{
			CAFile: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
		}
	}

	config := promtail.Config{
		Client:    clientConfig,
		BatchWait: 1 * time.Second,
		BatchSize: 1024 * 1024, // ~ 1 MB
		Timeout:   time.Second * 30,
		BackoffConfig: backoff.Config{
			MinBackoff: time.Second * 1,
			MaxBackoff: time.Second * 5,
			MaxRetries: 5,
		},
		URL:      flagext.URLValue{URL: URL},
		TenantID: tenantID,
	}

	client, err := promtail.New(
		promtail.NewMetrics(nil, nil),
		config,
		nil,
		kitlog.NewLogfmtLogger(os.Stdout),
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// SendLogWithPromtail creates an entry for the log using the Promtail API
func SendLogWithPromtail(client promtail.Client, log string, labels model.LabelSet) {
	client.Chan() <- api.Entry{
		Labels: labels,
		Entry:  logproto.Entry{Timestamp: time.Now(), Line: log},
	}
}
