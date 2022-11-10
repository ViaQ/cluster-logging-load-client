package loadclient

type Options struct {
	Command              commandType
	Threads              int
	LogLinesPerSec       int
	Destination          string
	Source               string
	SyntheticPayloadSize int
	TotalLogLines        int64
	LogFormat            string
	OutputFile           string
	DestinationAPIURL    string
	QueryFile            string
	Queries              []string
	QueryRange           string
	Loki                 Loki
	DisableSecurityCheck bool
}

type Loki struct {
	TenantID string
	Labels   string
}
