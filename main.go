package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/server"
)

var apiURL = flag.String("url", "", "send log via loki api using the provided url (e.g http://localhost:3100/api/prom/push)")
var logPerSec = flag.Int64("logps", 500, "The total amount of log per second to generate.(default 500)")

func init() {
	lvl := logging.Level{}
	if err := lvl.Set("debug"); err != nil {
		panic(err)
	}
	util.InitLogger(&server.Config{LogLevel: lvl})
	flag.Parse()
}

func main() {
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	if apiURL != nil && *apiURL != "" {
		logViaAPI(*apiURL, host)
		return
	}
	for {
		var out io.Writer
		var stream string
		switch rand.Intn(2) {
		case 1:
			out = os.Stderr
			stream = "stderr"
		default:
			out = os.Stdout
			stream = "stdout"

		}
		fmt.Fprintf(out, "ts=%s stream=%s host=%s lvl=%s msg=%s \n", time.Now().Format(time.RFC3339Nano), stream, host, randLevel(), randomLog())
		time.Sleep(time.Second / time.Duration(*logPerSec))
	}
}

func logViaAPI(apiURL string, hostname string) {
	u, err := url.Parse(apiURL)
	if err != nil {
		panic(err)
	}
	c, err := client.New(client.Config{
		BatchWait: 0,
		BatchSize: 100,
		Timeout:   time.Second * 30,
		BackoffConfig: util.BackoffConfig{
			MinBackoff: time.Second * 1,
			MaxBackoff: time.Second * 5,
			MaxRetries: 5,
		},
		URL: flagext.URLValue{URL: u},
	}, util.Logger)
	if err != nil {
		panic(err)
	}
	defer c.Stop()

	ticker := time.NewTicker(time.Second / time.Duration(*logPerSec))
	defer ticker.Stop()
	for {
		<-ticker.C
		_ = c.Handle(
			model.LabelSet{
				"hostname":  model.LabelValue(hostname),
				"service":   randService(),
				"level":     randLevel(),
				"component": randComponent(),
			}, time.Now(), randomLog())
	}
}

func randomLog() string {
	return loglines[rand.Intn(len(loglines))]
}

func randLevel() model.LabelValue {
	return levels[rand.Intn(4)]
}

func randComponent() model.LabelValue {
	return components[rand.Intn(5)]

}

func randService() model.LabelValue {
	return services[rand.Intn(6)]
}

var loglines = []string{
	"failing to cook potatoes",
	"successfully launched a car in space",
	"we got here",
	"panic: could not read the manual",
	"error while reading floppy disk",
	"failed to reach the cloud, try again on a rainy day",
	"failed to get an error message",
	"You're screwed !",
	"Oups I did it again",
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
	"container sinked in whales",
	"Don’t use beef stew as a computer password. It’s not stroganoff.",
	"I used stack overflow to fix this bug",
	"try googling this error message if it appears again",
	"change stuff and see what happens",
	"panic: this should never happen",
}

var levels = []model.LabelValue{
	"info",
	"warn",
	"debug",
	"error",
}

var components = []model.LabelValue{
	"devopsend",
	"fullstackend",
	"frontend",
	"everything-else",
	"backend",
}

var services = []model.LabelValue{
	"potatoes-cart",
	"phishing",
	"stateless-database",
	"random-policies-generator",
	"cookie-jar",
	"distributed-unicorn",
}
