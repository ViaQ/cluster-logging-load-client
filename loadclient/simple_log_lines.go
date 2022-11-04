package loadclient

import (
	"math/rand"

	"github.com/prometheus/common/model"
)

func randLevel() model.LabelValue {
	return levels[rand.Intn(4)]
}

func randComponent() model.LabelValue {
	return components[rand.Intn(5)]
}

func randService() model.LabelValue {
	return services[rand.Intn(6)]
}

var levels = []model.LabelValue{
	"info",
	"warn",
	"debug",
	"error",
}

var components = []model.LabelValue{
	"develop-send",
	"full-stack-end",
	"frontend",
	"everything-else",
	"backend",
}

var services = []model.LabelValue{
	"potatoes-cart",
	"phishing",
	"stateless-database",
	"random-policies-generator",
	"cookie-jar",
	"distributed-unicorn",
}
