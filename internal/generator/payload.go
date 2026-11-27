package generator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

const (
	// ConfigLogType generates messages from a user supplied payload config
	ConfigLogType LogType = "config"

	seqPlaceholder      = "{{SEQ}}"
	loaderIDPlaceholder = "{{LOADER_ID}}"
)

// PayloadTemplate is a weighted message template. Message may contain the
// placeholders {{SEQ}} and {{LOADER_ID}}.
type PayloadTemplate struct {
	Weight  float64 `json:"weight"`
	Message string  `json:"message"`
}

// PayloadConfig describes a weighted mix of message templates
type PayloadConfig struct {
	Seed      int               `json:"seed"`
	Templates []PayloadTemplate `json:"templates"`
}

// LoadPayloadConfig reads and validates a payload config file
func LoadPayloadConfig(path string) (*PayloadConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read payload config %s: %w", path, err)
	}
	cfg := &PayloadConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("unable to parse payload config %s: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid payload config %s: %w", path, err)
	}
	return cfg, nil
}

func (c *PayloadConfig) validate() error {
	if len(c.Templates) == 0 {
		return fmt.Errorf("at least one template is required")
	}
	var sum float64
	for i, t := range c.Templates {
		if t.Weight < 0 {
			return fmt.Errorf("template %d has a negative weight", i)
		}
		sum += t.Weight
	}
	if !(sum > 0) {
		return fmt.Errorf("sum of template weights must be greater than zero")
	}
	return nil
}

// PayloadSource produces messages from a PayloadConfig. It is not safe for
// concurrent use.
type PayloadSource struct {
	templates  []string
	cumulative []float64
	rng        *rand.Rand
	loaderID   string
	seq        int64
}

// NewPayloadSource creates a source that selects templates using the config seed
func NewPayloadSource(cfg *PayloadConfig, loaderID string) (*PayloadSource, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	s := &PayloadSource{
		rng:      rand.New(rand.NewSource(int64(cfg.Seed))),
		loaderID: loaderID,
	}
	var total float64
	for _, t := range cfg.Templates {
		total += t.Weight
		s.templates = append(s.templates, t.Message)
		s.cumulative = append(s.cumulative, total)
	}
	return s, nil
}

// Next returns the next message with placeholders substituted. The sequence
// number starts at 0 and increments once per message.
func (s *PayloadSource) Next() string {
	target := s.rng.Float64() * s.cumulative[len(s.cumulative)-1]
	idx := len(s.templates) - 1
	for i, c := range s.cumulative {
		if target < c {
			idx = i
			break
		}
	}
	msg := strings.NewReplacer(
		seqPlaceholder, strconv.FormatInt(s.seq, 10),
		loaderIDPlaceholder, s.loaderID,
	).Replace(s.templates[idx])
	s.seq++
	return msg
}
