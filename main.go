package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/grafana/loki/pkg/promtail/client"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/prometheus/common/model"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/weaveworks/common/logging"
	"github.com/weaveworks/common/server"
)

var (
	cfgFile    string
	apiURL     string
	logPerSec  int64
	remoteType string
	stopC      = make(chan os.Signal)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "logger",
	Short: "A log benchmark tool",
}

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "send randomly generated log to destination",
	Run: func(cmd *cobra.Command, args []string) {
		generateLog()
	},
}

// queryCmd represents the query command
var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "query the log storage",
	Run: func(cmd *cobra.Command, args []string) {
		q := viper.GetStringSlice("query")
		if len(q) == 0 {
			log.Fatal("Missing query in the config file")
		}
		queryLog(q)
	},
}

func init() {
	lvl := logging.Level{}
	if err := lvl.Set("debug"); err != nil {
		panic(err)
	}
	util.InitLogger(&server.Config{LogLevel: lvl})

	signal.Notify(stopC, os.Interrupt, syscall.SIGTERM)

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/logger.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "url", "", "send log via loki api using the provided url (e.g http://localhost:3100/api/prom/push)")
	rootCmd.PersistentFlags().Int64Var(&logPerSec, "logps", 500, "The total amount of log per second to generate.(default 500)")
	rootCmd.PersistentFlags().StringVar(&remoteType, "remote-type", "loki", "Type of the remote destination: loki, elasticsearch. (default loki)")

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(queryCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".logger" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".logger")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func queryLog(queries []string) {
	if apiURL != "" {
		switch remoteType {
		case "loki":
			fmt.Println("Query to loki: TO BE IMPLEMENTED")
		case "elasticsearch":
			fmt.Println("Query es")
			logQueryES(apiURL, queries)
		default:
			fmt.Printf("Unsupported remote type: %s\n", remoteType)
		}
		return
	}
}

func generateLog() {
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	if apiURL != "" {
		switch remoteType {
		case "loki":
			logViaAPI(apiURL, host)
		case "elasticsearch":
			fmt.Println("Sending logging to es")
			logViaEsCli(apiURL, host)
		default:
			fmt.Printf("Unsupported remote type: %s\n", remoteType)
		}
		return
	}

	ticker := time.NewTicker(time.Second / time.Duration(logPerSec))
	for {
		select {
		case <-stopC:
			ticker.Stop()
			return
		case <-ticker.C:
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
		}
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

	ticker := time.NewTicker(time.Second / time.Duration(logPerSec))
	defer ticker.Stop()
	for {
		select {
		case <-stopC:
			ticker.Stop()
			return
		case <-ticker.C:
			_ = c.Handle(
				model.LabelSet{
					"hostname":  model.LabelValue(hostname),
					"service":   randService(),
					"level":     randLevel(),
					"component": randComponent(),
				}, time.Now(), randomLog())
		}
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
