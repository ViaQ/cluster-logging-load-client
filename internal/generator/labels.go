package generator

import (
	"math/rand"

	"github.com/prometheus/common/model"
)

// LabelSetOptions describes which labels to include
type LabelSetOptions string

const (
	// ClientOnlyOption creates a label set with only the client label
	ClientOnlyOption LabelSetOptions = "client"

	// ClientHostOnlyOption creates a label set with only the client and host label
	ClientHostOnlyOption LabelSetOptions = "client-host"
)

var (
	components = []model.LabelValue{
		"develop-send",
		"full-stack-end",
		"frontend",
		"everything-else",
		"backend",
	}

	levels = []model.LabelValue{
		"info",
		"warn",
		"debug",
		"error",
	}

	services = []model.LabelValue{
		"potatoes-cart",
		"phishing",
		"stateless-database",
		"random-policies-generator",
		"cookie-jar",
		"distributed-unicorn",
	}

	streams = []model.LabelValue{
		"stderr",
		"stdout",
	}
)

// LogLabelSet creates a label set based on the configured options
func LogLabelSet(host string, options LabelSetOptions) model.LabelSet {
	switch options {
	case ClientOnlyOption:
		return model.LabelSet{
			"client": "promtail",
		}
	case ClientHostOnlyOption:
		return model.LabelSet{
			"client":   "promtail",
			"hostname": model.LabelValue(host),
		}
	default:
		return model.LabelSet{
			"client":    "promtail",
			"hostname":  model.LabelValue(host),
			"service":   randService(),
			"level":     randLevel(),
			"component": randComponent(),
		}
	}
}

func randLevel() model.LabelValue {
	return levels[rand.Intn(len(levels))]
}

func randComponent() model.LabelValue {
	return components[rand.Intn(len(components))]
}

func randService() model.LabelValue {
	return services[rand.Intn(len(services))]
}

func randStream() model.LabelValue {
	return streams[rand.Intn(len(streams))]
}
