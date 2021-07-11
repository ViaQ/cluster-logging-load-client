PHONY: all build test clean build-image deploy undeploy push-image run-es
.DEFAULT_GOAL := all

include .bingo/Variables.mk

IMAGE_PREFIX ?= quay.io/openshift-logging
IMAGE_TAG := 0.1
ES_CONTAINER_NAME=elasticsearch
ES_IMAGE_TAG=docker.io/library/elasticsearch:6.8.12

all: test build-image

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

run-es:
	podman run -d --name $(ES_CONTAINER_NAME) \
		-p 9200:9200 -p 9300:9300 \
		-e "discovery.type=single-node" \
		$(ES_IMAGE_TAG)
