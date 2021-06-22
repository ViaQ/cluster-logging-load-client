package loadclient

import (
	"github.com/prometheus/common/model"
	"math/rand"
)

func getSimpleLogLine() string {
	return simpleLogLines[rand.Intn(len(simpleLogLines))]
}

func randLevel() model.LabelValue {
	return levels[rand.Intn(4)]
}

func randComponent() model.LabelValue {
	return components[rand.Intn(5)]
}

func randService() model.LabelValue {
	return services[rand.Intn(6)]
}

var simpleLogLines = []string{
	"failing to cook potatoes",
	"successfully launched a car in space",
	"we got here",
	"panic: could not read the manual",
	"error while reading floppy disk",
	"failed to reach the cloud, try again on a rainy day",
	"failed to get an error message",
	"You're screwed !",
	"I did it again",
	"a chicken died during processing",
	"sorry the server is not in a mood",
	"Stupidity made this error, not me",
	"random error happened during compression",
	"too many foobar variable",
	"cannot over-write a locked variable.",
	"foo insists on strongly-typed programming languages",
	"John Doe solved the Travelling Salesman problem in O(1) time. Here's the pseudo-code: Break salesman into N pieces. Kick each piece to a different city.",
	"infinite loop succeeded in less than 3 seconds",
	"could not compute the last digit of PI",
	"OS not found try installing one",
	"container sink in whales",
	"Don’t use beef stew as a computer password. It’s not stroganoff.",
	"I used stack overflow to fix this bug",
	"try googling this error message if it appears again",
	"change stuff and see what happens",
	"panic: this should never happen",
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

