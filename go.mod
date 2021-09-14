module github.com/ViaQ/cluster-logging-load-client

go 1.16

require (
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/cortexproject/cortex v1.2.1-0.20200803161316-7014ff11ed70
	github.com/elastic/go-elasticsearch/v6 v6.8.10
	github.com/go-kit/kit v0.10.0
	github.com/gogo/googleapis v1.2.0 // indirect
	github.com/gogo/status v1.1.0 // indirect
	github.com/grafana/loki v1.6.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/prometheus/common v0.10.0
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	google.golang.org/genproto v0.0.0-20200911024640-645f7a48b24f // indirect
	google.golang.org/grpc v1.31.1 // indirect
	gopkg.in/yaml.v2 v2.3.0
)

replace k8s.io/client-go => k8s.io/client-go v0.20.4
