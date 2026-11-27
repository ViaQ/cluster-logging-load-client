package internal

import "time"

type Options struct {
	Command              string
	Destination          string
	OutputFile           string
	ClientURL            string
	DisableSecurityCheck bool
	LogsPerSecond        int
	LogType              string
	LogFormat            string
	LabelType            string
	SyntheticPayloadSize int
	UseRandomHostname    bool
	Tenant               string
	QueriesPerMinute     int
	Query                string
	QueryRange           string
	PayloadConfig        string
	LoaderID             string
	StartSignalFile      string
	Duration             time.Duration
	SentCountFile        string
}
