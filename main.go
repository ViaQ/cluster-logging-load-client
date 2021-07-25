package main

import (
	"encoding/json"
	"fmt"
	"github.com/ViaQ/cluster-logging-load-client/loadclient"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"strings"
	"time"
)

var opt = loadclient.Options{}
var cfgFile string
var logLevel string
var envPrefix = "LOADCLIENT"

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
		opt.Command = loadclient.Generate
		loadclient.GenerateLog(opt)
	},
}

// queryCmd represents the query command
var queryCmd = &cobra.Command{

	Use:   "query",
	Short: "query the log storage",
	Run: func(cmd *cobra.Command, args []string) {
		logConfig()
		opt.Command = loadclient.Query
		loadclient.QueryLog(opt)
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
	rootCmd.PersistentFlags().Int64Var(&opt.LogLinesPerSec, "log-lines-rate", 1, "The total amount of log lines per thread per second to generate.(default 1)")
	rootCmd.PersistentFlags().StringVar(&opt.Source, "source", "simple", "Log lines Source: simple, application, synthetic. (default simple)")
	rootCmd.PersistentFlags().StringVar(&opt.Destination, "destination", "stdout", "Log Destination: loki, elasticsearch, stdout, file. (default stdout)")
	rootCmd.PersistentFlags().Int64Var(&opt.TotalLogLines, "totalLogLines", 0, "Total number of log lines per thread (default 0 - infinite)")

	rootCmd.PersistentFlags().StringVar(&opt.LogFormat, "output-format", "default", "The output format: default, crio (mimic CRIO output), csv")
	rootCmd.PersistentFlags().IntVar(&opt.SyntheticPayloadSize, "synthetic-payload-size", 100, "Payload length [int] (default = 100)")
	rootCmd.PersistentFlags().StringVar(&opt.OutputFile, "file", "output", "The file to output (default: output)")
	rootCmd.PersistentFlags().StringVar(&opt.DestinationAPIURL, "destination-url", "", "send logs via api using the provided url (e.g http://localhost:3100/api/prom/push)")
	rootCmd.PersistentFlags().StringVar(&opt.DestinationAPIURL, "url", "", "Alt. destination flag (see --destination-url)")
	rootCmd.PersistentFlags().StringVar(&opt.Loki.TenantID, "loki-tenant-ID", "", "Loki tenantID (default = fake)")
	rootCmd.PersistentFlags().StringVar(&opt.Loki.TenantID, "tenant", "fake", "Alt. Loki tenantID flag (see --loki-tenant-ID)")
	rootCmd.PersistentFlags().StringVar(&opt.Loki.Labels, "loki-labels", "random", "Loki labels: none,host,random (default = random)")

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "error", "Log level: debug, info, warning, error (default = error)")

	rootCmd.PersistentFlags().StringArrayVar(&opt.Queries, "queries", []string{}, "list of queries e.g. {client=\"promtail\"} (default = none)")
	rootCmd.PersistentFlags().StringVar(&opt.QueryFile, "query-file", "", "Query file name (default = none)")

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(queryCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	v := viper.New()

	if cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".logger" (without extension).
		v.AddConfigPath(home)
		v.SetConfigName(".logger")
	}

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := v.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", v.ConfigFileUsed())
	}

	bindFlags(rootCmd, v)

	ll, err := log.ParseLevel(logLevel)
	if err != nil {
		ll = log.ErrorLevel
	}
	log.SetLevel(ll)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			_ = v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

func main() {
	rand.Seed(time.Now().UnixNano())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
