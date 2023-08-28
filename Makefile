PHONY: all build test clean build-image deploy undeploy push-image run-es
.DEFAULT_GOAL := help

include .bingo/Variables.mk

REGISTRY_ORG ?= openshift-logging
VERSION ?= 0.1

IMG ?= quay.io/$(REGISTRY_ORG)/cluster-logging-load-client:$(VERSION)

ES_CONTAINER_NAME=elasticsearch
ES_IMAGE_TAG=docker.io/library/elasticsearch:6.8.12
LOKI_CONTAINER_NAME=loki
LOKI_IMAGE_TAG=docker.io/grafana/loki:2.8.3

OCI_RUNTIME ?= $(shell which podman || which docker)

all: clean build test build-image local ## Runs all commands

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

lint: $(GOLINT) ## Lints the files
	$(GOLINT) main.go

fmt: $(GOFUMPT)
	$(GOFUMPT) -w .

build: lint fmt ## Build the executable
	go build -o logger -v main.go

test: lint fmt ## Run the tests
	go test ./...

clean: ## Delete the object files and cached files including the executable
	rm -f ./logger
	go clean ./...

build-image: ## Build the image
	$(OCI_RUNTIME) build -t $(IMG) .

push-image: ## Push the image
	$(OCI_RUNTIME) push $(IMG)

deploy: ## Deploy the image
	kubectl apply -f deployment.yaml

undeploy: ## Undeploy the image
	kubectl delete -f deployment.yaml

local: clean-local deploy-local-es deploy-local-loki run-local-es-generate run-local-es-query run-local-loki-generate run-local-loki-query ## Run all the local commands

clean-local: ## Clean all the local containers
	podman kill $(ES_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman rm $(ES_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman kill $(LOKI_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman rm $(LOKI_CONTAINER_NAME) > /dev/null 2>&1 || true

run-local-es-generate: ## Run logger with remote type elasticsearch
	./logger --command generate --log-level info --destination elasticsearch --url http://localhost:9200/

run-local-es-query: ## Generate query requests to elasticsearch v6
	./logger --command query --log-level info --destination elasticsearch --url http://localhost:9200/ --query '{ "query": { "range": { "created_at": { "time_zone": "UTC", "gte": "now-1h/h", "lt": "now" } } } }'

run-local-loki-generate: ## Run logger with remote type loki
	./logger --command generate --log-level info --destination loki --url http://localhost:3100/api/prom/push

run-local-loki-query: ## Generate query requests to loki
	./logger --command query --log-level info --destination loki --url http://localhost:3100 --query {client="promtail"}

deploy-local-es: ## Launch an elasticsearch container
	podman run -d --name $(ES_CONTAINER_NAME) \
		-p 9200:9200 -p 9300:9300 \
		-e "discovery.type=single-node" \
		$(ES_IMAGE_TAG)
	sleep 20

deploy-local-loki: ## Launch a loki container
	podman run -d --name $(LOKI_CONTAINER_NAME) \
		-p 3100:3100 \
		-e "discovery.type=single-node" \
		$(LOKI_IMAGE_TAG)
	sleep 20
