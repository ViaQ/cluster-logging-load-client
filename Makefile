PHONY: all build test clean build-image deploy undeploy push-image run-es
.DEFAULT_GOAL := all

include .bingo/Variables.mk

IMAGE_PREFIX ?= quay.io/openshift-logging
IMAGE_TAG := 0.1
ES_CONTAINER_NAME=elasticsearch
ES_IMAGE_TAG=docker.io/library/elasticsearch:6.8.12
LOKI_CONTAINER_NAME=loki
LOKI_IMAGE_TAG=docker.io/grafana/loki:2.2.1

all: clean build test build-image local

lint: $(GOLINT)
	 $(GOLINT) main.go

build: lint
	go build -o logger -v main.go

test: lint
	go test -v loadclient/*.go

clean:
	rm -f ./logger
	go clean ./...

build-image:
	docker build -t $(IMAGE_PREFIX)/cluster-logging-load-client .
	docker tag $(IMAGE_PREFIX)/cluster-logging-load-client $(IMAGE_PREFIX)/cluster-logging-load-client:$(IMAGE_TAG)

push-image:
	docker push $(IMAGE_PREFIX)/cluster-logging-load-client:$(IMAGE_TAG)
	docker push $(IMAGE_PREFIX)/cluster-logging-load-client:latest

deploy:
	kubectl apply -f deployment.yaml

undeploy:
	kubectl delete -f deployment.yaml

local: clean-local deploy-local-es deploy-local-loki run-local-es-generate run-local-es-query run-local-loki-generate run-local-loki-query

clean-local:
	podman kill $(ES_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman rm $(ES_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman kill $(LOKI_CONTAINER_NAME) > /dev/null 2>&1 || true
	podman rm $(LOKI_CONTAINER_NAME) > /dev/null 2>&1 || true

run-local-es-generate:
	./logger generate --log-level info --destination elasticsearch --destination-url http://localhost:9200/ --totalLogLines 5

run-local-es-query:
	./logger query --log-level info --destination elasticsearch --destination-url http://localhost:9200/ --query-file ./config/es_queries.yaml --totalLogLines 2

run-local-loki-generate:
	./logger generate --log-level info --destination loki --destination-url http://localhost:3100/api/prom/push --totalLogLines 5

run-local-loki-query:
	./logger query --log-level info --destination loki --destination-url http://localhost:3100 --query-file ./config/loki_queries.yaml --totalLogLines 2

deploy-local-es:
	podman run -d --name $(ES_CONTAINER_NAME) \
		-p 9200:9200 -p 9300:9300 \
		-e "discovery.type=single-node" \
		$(ES_IMAGE_TAG)
	sleep 20

deploy-local-loki:
	podman run -d --name $(LOKI_CONTAINER_NAME) \
		-p 3100:3100 \
		-e "discovery.type=single-node" \
		$(LOKI_IMAGE_TAG)
	sleep 20
