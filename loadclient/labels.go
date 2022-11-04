import (
	"math/rand"

	"github.com/prometheus/common/model"
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