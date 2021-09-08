PHONY: all build test clean build-image deploy undeploy push-image run-es
.DEFAULT_GOAL := all

include .bingo/Variables.mk

IMAGE_PREFIX ?= quay.io/openshift-logging
IMAGE_TAG := 0.1
ES_CONTAINER_NAME=elasticsearch
ES_IMAGE_TAG=docker.io/library/elasticsearch:6.8.12
LOKI_CONTAINER_NAME=loki
LOKI_IMAGE_TAG=docker.io/grafana/loki:2.2.1

##@ <target>:
all: ## Run everything (clean, build, test...)
	clean build test build-image local

lint: $(GOLINT)
	 $(GOLINT) main.go

build: ## Build the executable
	make lint
	go build -o logger -v main.go

test: ## Run the tests
	make lint
	go test -v loadclient/*.go

clean: ## Delete the object files and cached files including the executable "logger"
	rm -f ./logger
	go clean ./...

build-image: ## Build the image
	docker build -t $(IMAGE_PREFIX)/cluster-logging-load-client .
	docker tag $(IMAGE_PREFIX)/cluster-logging-load-client $(IMAGE_PREFIX)/cluster-logging-load-client:$(IMAGE_TAG)

push-image: ## Push the image
	docker push $(IMAGE_PREFIX)/cluster-logging-load-client:$(IMAGE_TAG)
	docker push $(IMAGE_PREFIX)/cluster-logging-load-client:latest

deploy: ## Deploy the image (build-image must be called before deploy)
	kubectl apply -f deployment.yaml

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

undeploy: ## Undeploy the image
	kubectl delete -f deployment.yaml

local: ## Run all the local commands
	clean-local deploy-local-es deploy-local-loki run-local-es-generate run-local-es-query run-local-loki-generate run-local-loki-query

clean-local: ## Clean all the local containers
	podman kill $(ES_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman rm $(ES_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman kill $(LOKI_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman rm $(LOKI_CONTAINER_NAME) > /dev/null 2>&1 || true

run-local-es-generate: ## Run logger and with remote type elasticsearch
	./logger generate --log-level info --destination elasticsearch --destination-url http://localhost:9200/ --totalLogLines 5

run-local-es-query: ## Generate query requests to elasticsearch v6
	./logger query --log-level info --destination elasticsearch --destination-url http://localhost:9200/ --query-file ./config/es_queries.yaml --totalLogLines 2

run-local-loki-generate: ## Run logger and set with remote type loki
	./logger generate --log-level info --destination loki --destination-url http://localhost:3100/api/prom/push --totalLogLines 5

run-local-loki-query: ## Generate query requests to loki
	./logger query --log-level info --destination loki --destination-url http://localhost:3100 --query-file ./config/loki_queries.yaml --totalLogLines 2

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
