package generator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// LogType describes the type of generated log
type LogType string

const (
	// ApplicationLogType represents a log that is likely to be seen in an
	// application runtime environment.
	ApplicationLogType LogType = "application"

	// SyntheticLogType represents a log that is composed of random
	// alphabetical characters of a certain size.
	SyntheticLogType LogType = "synthetic"
)

// ElasticsearchLogContent describes the json content for logs for Elasticsearch
type ElasticsearchLogContent struct {
	Hostname  string    `json:"hostname"`
	Service   string    `json:"service"`
	Level     string    `json:"level"`
	Component string    `json:"component"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	// SyntheticSampleSelection is a string of characters a SyntheticLogType will use
	// to create a synthetic log.
	SyntheticSampleSelection = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	simpleSamples = []string{
		"failing to cook potatoes",
		"successfully launched a car in space",
		"we got here",
		"panic: could not read the manual",
		"error while reading floppy disk",
		"failed to reach the cloud, try again on a rainy day",
		"failed to get an error message",
		"You're screwed !",
		"I did it again",
		"a chicken died during processing",
		"sorry the server is not in a mood",
		"Stupidity made this error, not me",
		"random error happened during compression",
		"too many foobar variable",
		"cannot over-write a locked variable.",
		"foo insists on strongly-typed programming languages",
		"John Doe solved the Travelling Salesman problem in O(1) time. Here's the pseudo-code: Break salesman into N pieces. Kick each piece to a different city.",
		"infinite loop succeeded in less than 3 seconds",
		"could not compute the last digit of PI",
		"OS not found try installing one",
		"container sink in whales",
		"Don’t use beef stew as a computer password. It’s not stroganoff.",
		"I used stack overflow to fix this bug",
		"try googling this error message if it appears again",
		"change stuff and see what happens",
		"panic: this should never happen",
	}

	applicationSamples = []string{
		"Stderr: 'Get Auth Capabilities error\\nError issuing Get Channel Authentication Capabilities request\\nError: Unable to establish IPMI v2 / RMCP+ session\\n': oslo_concurrency.processutils.ProcessExecutionError: Unexpected error while running command.",
		"2021/04/27 02:46:30 http: proxy error: context canceled",
		"time=\"2021-04-27T01:24:13Z\" level=error msg=\"[sync] error checking for updates [openshift-marketplace/certified-operators] - Get https://quay.io/cnr/api/v1/packages?namespace=certified-operators: context deadline exceeded\"",
		"I0427 01:14:33.934764       1 log.go:172] http: TLS handshake error from 69.101.110.80:45106: EOF",
		"2021-04-27 02:06:24.986 1 WARNING ironic.conductor.manager [req-21584cf6-35d0-45ac-b2c8-0d3c26d5b483 - - - - -] During sync_power_state, could not get power state for node 954c6cbf-d8fe-495f-b91f-de0d038b28ce, attempt 1 of 3. Error: IPMI call failed: power status..: ironic.common.exception.IPMIFailure: IPMI call failed: power status.",
		"E0427 02:47:01.619035       1 authentication.go:53] Unable to authenticate the request due to an error: [invalid bearer token, context canceled]",
		"2021-04-27T00:12:56.175Z|00007|jsonrpc|WARN|unix#61: receive error: Connection reset by peer",
		"{\"level\":\"error\",\"ts\":1619482401.1593049,\"logger\":\"controller-runtime.manager\",\"msg\":\"Failed to get API Group-Resources\",\"error\":\"Get https://1.1.1.1:443/api?timeout=32s: dial tcp 1.23.0.1:443: i/o timeout\",\"stacktrace\":\"github.com/go-logr/zapr.(*zapLogger).Error\\n\\t/go/src/github.com/nmstate/kubernetes-nmstate/vendor/github.com/go-logr/zapr/zapr.go:128\\nsigs.k8s.io/controller-runtime/pkg/manager.New\\n\\t/go/src/github.com/nmstate/kubernetes-nmstate/vendor/sigs.k8s.io/controller-runtime/pkg/manager/manager.go:238\\nmain.main\\n\\t/go/src/github.com/nmstate/kubernetes-nmstate/cmd/manager/main.go:124\\nruntime.main\\n\\t/usr/lib/golang/src/runtime/proc.go:203\"",
		"I0427 01:09:55.209303       1 main.go:218] Error syncing csr csr-2cxlq: CSR csr-2cxlq for node client cert has wrong user system:node:worker-01 or groups map[system:authenticated:{} system:nodes:{}]",
		"{\"level\":\"error\",\"ts\":1619529878.138428,\"logger\":\"controller-runtime.controller\",\"msg\":\"Reconciler error\",\"controller\":\"hyperconverged-controller\",\"request\":\"openshift-cnv/kubevirt-hyperconverged\",\"error\":\"Operation cannot be fulfilled on hyperconvergeds.hco.kubevirt.io \"kubevirt-hyperconverged\": the object has been modified; please apply your changes to the latest version and try again\",\"stacktrace\":\"github.com/go-logr/zapr.(*zapLogger).Error\\n\\t/go/src/github.com/kubevirt/hyperconverged-cluster-operator/vendor/github.com/go-logr/zapr/zapr.go:128\\nsigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).reconcileHandler\\n\\t/go/src/github.com/kubevirt/hyperconverged-cluster-operator/vendor/sigs.k8s.io/controller-runtime/pkg/internal/controller/controller.go:258\\nsigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).processNextWorkItem\\n\\t/go/src/github.com/kubevirt/hyperconverged-cluster-operator/vendor/sigs.k8s.io/controller-runtime/pkg/internal/controller/controller.go:232\\nsigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).worker\\n\\t/go/src/github.com/kubevirt/hyperconverged-cluster-operator/vendor/sigs.k8s.io/controller-runtime/pkg/internal/controller/controller.go:211\\nk8s.io/apimachinery/pkg/util/wait.JitterUntil.func1\\n\\t/go/src/github.com/kubevirt/hyperconverged-cluster-operator/vendor/k8s.io/apimachinery/pkg/util/wait/wait.go:152\\nk8s.io/apimachinery/pkg/util/wait.JitterUntil\\n\\t/go/src/github.com/kubevirt/hyperconverged-cluster-operator/vendor/k8s.io/apimachinery/pkg/util/wait/wait.go:153\\nk8s.io/apimachinery/pkg/util/wait.Until\\n\\t/go/src/github.com/kubevirt/hyperconverged-cluster-operator/vendor/k8s.io/apimachinery/pkg/util/wait/wait.go:88\"}",
		"[localhost]: FAILED! => {\"msg\": \"An unhandled exception occurred while running the lookup plugin 'k8s'. Error was a <class 'ansible.errors.AnsibleError'>, original message: Failed to find exact match for kubevirt.io/v1.KubevirtCommonTemplatesBundle by [kind, name, singularName, shortNames]\"}",
		"E0427 11:44:58.439709       1 memcache.go:206] couldn't get resource list for metrics.k8s.io/v1beta1: an error on the server",
		"2021-04-27 11:39:43.113686 I | embed: rejected connection from \"1.1.1.1:37074\" (error \"EOF\", ServerName)",
		"W0427 11:39:43.113579       1 clientconn.go:1208] grpc: addrConn.createTransport failed to connect to {https://1.1.1.1:2379  <nil> 0 <nil>}. Err :connection error: desc = transport: authentication handshake failed: context canceled. Reconnecting...",
		"2021-04-27T09:24:51Z|00650|stream_ssl|WARN|SSL_write: system error (Connection reset by peer)",
		"E0427 16:50:08.356649       1 status.go:71] apiserver received an error that is not an metav1.Status: &errors.errorString{s:\"context canceled\"}",
		"E0427 20:06:06.185567       1 limiter.go:165] error reloading router: wait: no child processes",
		"E0412 21:37:23.019649       1 proxy.go:73] Unable to authenticate the request due to an error: Post https://1.1.1.1:443/apis/authentication.k8s.io/v1/tokenreviews: context canceled",
		"[ERROR] plugin/errors: 2 kubernetes.default.svc. A: read udp 1.1.1.1:54608->11.1.1.1:53: i/o timeout",
		"E0426 20:39:28.065697       1 scheduler.go:599] error selecting node for pod: running \"VolumeBinding\" filter plugin for pod \"eric-data-document-database-pg-1\": pod has unbound immediate PersistentVolumeClaims",
		"Warning: failed to query journal: Bad message (os error 74)",
		"E0427 02:54:04.283531    8550",
		"[DEBUG] plugin/errors: 2 kubernetes.default.svc. A: read udp 1.1.1.1:54608->1.1.1.1:53: i/o timeout",
		"D0426 20:39:28.065697       1 scheduler.go:599] error selecting node for pod: running \"VolumeBinding\" filter plugin for pod \"eric-data-document-database-pg-1\": pod has unbound immediate PersistentVolumeClaims",
	}
)

// RandomLog returns a log of a given type from the requested sample set.
func RandomLog(logType LogType, logSize int) (string, error) {
	switch logType {
	case ApplicationLogType:
		index := rand.Intn(len(applicationSamples))
		return applicationSamples[index], nil
	case SyntheticLogType:
		if logSize < 0 {
			return "", fmt.Errorf("invalid size for sythentic log")
		}
		return generateSyntheticLog(logSize), nil
	default:
		index := rand.Intn(len(simpleSamples))
		return simpleSamples[index], nil
	}
}

// NewElasticsearchLogContent returns a byte array representing the json content for
// a log to be consumed by Elasticsearch.
func NewElasticsearchLogContent(host, logLine string) ([]byte, error) {
	content := ElasticsearchLogContent{
		Hostname:  host,
		Service:   string(randService()),
		Level:     string(randLevel()),
		Component: string(randComponent()),
		Body:      logLine,
		CreatedAt: time.Now().Round(time.Second).UTC(),
	}

	data, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("error encoding elasticsearch log (%s): %s", logLine, err)
	}
	return data, nil
}

func generateSyntheticLog(size int) string {
	sampleSize := len(SyntheticSampleSelection)

	var builder strings.Builder
	for i := 0; i < size; i++ {
		builder.WriteByte(SyntheticSampleSelection[rand.Intn(sampleSize)])
	}
	return builder.String()
}
