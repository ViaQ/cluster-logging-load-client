package loadclient

type Options struct {
	Threads              int
	LogLinesPerSec       int
	Destination          string
	Source               string
	SyntheticPayloadSize int
	TotalLogLines        int
	LogFormat            string
	OutputFile           string
	DestinationAPIURL    string
	QueryFile            string
	Loki                 Loki
}

type Loki struct {
	TenantID string
	Labels   string
}
