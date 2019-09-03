package main

import (
	"math/rand"
	"net/url"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/grafana/loki/pkg/promtail/client"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/server"
)

func init() {
	lvl := logging.Level{}
	if err := lvl.Set("debug"); err != nil {
		panic(err)
	}
	util.InitLogger(&server.Config{
		LogLevel: lvl,
	})
}

func main() {
	u, err := url.Parse("http://localhost:3100/api/prom/push")
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
	for {
		_ = c.Handle(
			model.LabelSet{
				"service":   randService(),
				"level":     randLevel(),
				"component": randComponent(),
			}, time.Now(), randomLog())
		time.Sleep(time.Millisecond * 100)
	}

}

func randomLog() string {
	return loglines[rand.Intn(len(loglines))]
}

var loglines = []string{
	"failing to cook potatoes",
	"sucessfully launched a car in space",
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

func randLevel() model.LabelValue {
	switch rand.Intn(3) {
	case 0:
		return "info"
	case 1:
		return "warn"
	case 2:
		return "debug"
	default:
		return "error"
	}
}

func randComponent() model.LabelValue {
	switch rand.Intn(4) {
	case 0:
		return "devopsend"
	case 1:
		return "fullstackend"
	case 2:
		return "frontend"
	case 3:
		return "everything-else"
	default:
		return "backend"
	}
}

func randService() model.LabelValue {
	switch rand.Intn(5) {
	case 0:
		return "potatoes-cart"
	case 1:
		return "phishing"
	case 2:
		return "stateless-database"
	case 3:
		return "random-policies-generator"
	case 4:
		return "cookie-jar"
	default:
		return "distributed-unicorn"
	}
}
