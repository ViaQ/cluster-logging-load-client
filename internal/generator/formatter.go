package generator

import (
	"encoding/json"
	"fmt"
	"time"
)

// Format describes the way a log should be formatted
type Format string

const (
	// CRIOFormat formats a log to appear in CRIO style
	CRIOFormat Format = "crio"

	// CRIOFormat formats a log to appear in CSV style
	CSVFormat Format = "csv"

	// JSONFormat formats a log to appear in JSON style
	JSONFormat Format = "json"
)

func FormatLog(style Format, hash string, messageCount int64, payload string) (string, error) {
	now := time.Now().Format(time.RFC3339Nano)

	switch style {
	case CRIOFormat:
		return fmt.Sprintf("%s stdout F goloader seq - %s - %010d - %s\n", now, hash, messageCount, payload), nil
	case CSVFormat:
		return fmt.Sprintf("ts=%s stream=%s host=%s level=%s count=%d msg=%q\n", now, randStream(), hash, randLevel(), messageCount, payload), nil
	case JSONFormat:
		message := map[string]interface{}{
			"ts":     now,
			"stream": randStream(),
			"host":   hash,
			"lvl":    randLevel(),
			"count":  messageCount,
			"msg":    payload,
		}
		messageJSON, err := json.Marshal(message)
		if err != nil {
			return "", err
		}
		return fmt.Sprintln(string(messageJSON)), nil
	default:
		return fmt.Sprintf("goloader seq - %s - %010d - %s\n", hash, messageCount, payload), nil
	}
}
