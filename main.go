package main

import (
	"encoding/json"
	"fmt"
	"github.com/ViaQ/cluster-logging-load-client/loadclient"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"time"
)

var opt = loadclient.Options{}
var cfgFile string
var logLevel string

// rootCmd represents the root command
var rootCmd = &cobra.Command{
	Use:   "logger",
	Short: "A log benchmark tool",
}

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "send randomly generated log lines to Destination",
	Run: func(cmd *cobra.Command, args []string) {
		logConfig()
		loadclient.GenerateLog(opt)
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
		logConfig()
		loadclient.QueryLog(q,opt)
	},
}

func logConfig() {
	configAsJSON, _ := json.MarshalIndent(opt, "", "\t")
	log.Infof("configuration:\n%s\n", configAsJSON)
}


func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/logger.yaml)")
	rootCmd.PersistentFlags().IntVar(&opt.Threads, "threads", 1, "Number of threads.(default 1)")
	rootCmd.PersistentFlags().IntVar(&opt.LogLinesPerSec, "log-lines-rate", 1, "The total amount of log lines per thread per second to generate.(default 1)")
	rootCmd.PersistentFlags().StringVar(&opt.Source, "source", "simple", "Log lines Source: simple, application, synthetic. (default simple)")
	rootCmd.PersistentFlags().StringVar(&opt.Destination, "destination", "stdout", "Log Destination: loki, elasticsearch, stdout, file. (default stdout)")
	rootCmd.PersistentFlags().IntVar(&opt.TotalLogLines, "totalLogLines", 0, "Total number of log lines per thread (default 0 - infinite)")

	rootCmd.PersistentFlags().StringVar(&opt.LogFormat, "output-format", "default", "The output format: default, crio (mimic CRIO output), csv")
	rootCmd.PersistentFlags().IntVar(&opt.SyntheticPayloadSize, "synthetic-payload-size", 100, "Payload length [int] (default = 100)")
	rootCmd.PersistentFlags().StringVar(&opt.OutputFile, "file", "output", "The file to output (default: output)")
	rootCmd.PersistentFlags().StringVar(&opt.DestinationAPIURL, "destination-url", "", "send logs via api using the provided url (e.g http://localhost:3100/api/prom/push)")
	rootCmd.PersistentFlags().StringVar(&opt.Loki.TenantID, "loki-tenant-ID", "fake", "Loki tenantID (default = fake)")
	rootCmd.PersistentFlags().StringVar(&opt.Loki.Labels, "loki-labels", "random", "Loki labels: none,host,random (default = random)")

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "error", "Log level: debug, info, warning, error (default = error)")

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

	ll, err := log.ParseLevel(logLevel)
	if err != nil {
		ll = log.ErrorLevel
	}
	log.SetLevel(ll)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})}

func main() {
	rand.Seed(time.Now().UnixNano())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
