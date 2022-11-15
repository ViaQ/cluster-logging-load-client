package internal

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
	Tenant               string
	QueriesPerMinute     int
	Query                string
	QueryRange           string
}
